package ideachat

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"github.com/kaos-control/kaos-control/internal/artifact"
)

// slugRe is the valid slug pattern: lowercase alphanumeric with internal hyphens.
var slugRe = regexp.MustCompile(`^[a-z0-9][a-z0-9\-]*[a-z0-9]$|^[a-z0-9]$`)

// Response is the result of a single Converse call.
type Response struct {
	Reply       string
	Status      string // StatusConversing | StatusProposed
	ProposedSlug string
	ProposedFM  *artifact.Frontmatter
	ProposedBody string
}

// llmAction is the structured JSON the LLM is expected to return.
type llmAction struct {
	Action string   `json:"action"` // "clarify" | "propose"
	Reply  string   `json:"reply"`
	Slug   string   `json:"slug"`
	Title  string   `json:"title"`
	Labels []string `json:"labels"`
	Body   string   `json:"body"`
}

// Converse takes a user message, updates the session, calls the LLM, and
// returns a Response. existingLabels constrains label choice. existingSlugs
// is used for collision detection. modelCfg controls the model and prompt.
func Converse(
	ctx context.Context,
	sess *Session,
	userMsg string,
	existingLabels []string,
	existingSlugs []string,
	modelCfg ModelConfig,
) (*Response, error) {
	// Append the new user message to history.
	sess.Messages = append(sess.Messages, Message{Role: "user", Content: userMsg})

	// Build the LLM message list.
	llmMsgs := buildLLMMessages(sess)

	// Call the LLM.
	raw, err := CallLLM(ctx, modelCfg, llmMsgs)
	if err != nil {
		return nil, fmt.Errorf("Converse: LLM call failed: %w", err)
	}

	// Parse the structured JSON response.
	action, parseErr := parseAction(raw)
	if parseErr != nil {
		// If parsing fails, treat the raw text as a clarifying reply.
		sess.Messages = append(sess.Messages, Message{Role: "assistant", Content: raw})
		return &Response{Reply: raw, Status: StatusConversing}, nil
	}

	// Append the assistant turn.
	sess.Messages = append(sess.Messages, Message{Role: "assistant", Content: raw})

	// If we've hit the clarification limit, force a proposal on the next turn
	// by signalling the status; if the model still clarifies, bump the counter.
	switch action.Action {
	case "clarify":
		sess.ClarifyCount++
		return &Response{Reply: action.Reply, Status: StatusConversing}, nil

	case "propose":
		slug, err := resolveSlug(ctx, action.Slug, existingSlugs, modelCfg, sess)
		if err != nil {
			return nil, fmt.Errorf("Converse: resolving slug: %w", err)
		}

		// Constrain labels to existing vocabulary.
		labels := filterLabels(action.Labels, existingLabels)

		fm := artifact.Frontmatter{
			Title:   action.Title,
			Type:    "idea",
			Status:  "draft",
			Lineage: slug,
			Labels:  labels,
		}

		sess.Status = StatusProposed
		sess.ProposedSlug = slug
		sess.ProposedFM = fm
		sess.ProposedBody = action.Body

		return &Response{
			Reply:        action.Reply,
			Status:       StatusProposed,
			ProposedSlug: slug,
			ProposedFM:   &fm,
			ProposedBody: action.Body,
		}, nil

	default:
		// Unknown action – treat as clarification.
		sess.ClarifyCount++
		return &Response{Reply: action.Reply, Status: StatusConversing}, nil
	}
}

// buildLLMMessages converts session history to the LLM message format.
// When ClarifyCount >= 3, a system-level instruction is prepended as the
// first user message to force a proposal.
func buildLLMMessages(sess *Session) []LLMMessage {
	msgs := make([]LLMMessage, 0, len(sess.Messages))
	for _, m := range sess.Messages {
		msgs = append(msgs, LLMMessage(m))
	}

	// If we've reached the clarification limit, inject a forcing instruction
	// before the final user message.
	if sess.ClarifyCount >= 3 && len(msgs) > 0 {
		last := msgs[len(msgs)-1]
		forcing := LLMMessage{
			Role:    "user",
			Content: last.Content + "\n\n[SYSTEM: You have asked enough clarifying questions. You MUST now produce a proposal with action=propose.]",
		}
		msgs[len(msgs)-1] = forcing
	}
	return msgs
}

// parseAction extracts the llmAction from the raw LLM reply.
// The LLM is expected to return a JSON block anywhere in its response.
func parseAction(raw string) (*llmAction, error) {
	// Look for a JSON block delimited by ```json ... ``` or a bare { ... }.
	jsonStr := extractJSON(raw)
	if jsonStr == "" {
		return nil, fmt.Errorf("no JSON found in response")
	}
	var a llmAction
	if err := json.Unmarshal([]byte(jsonStr), &a); err != nil {
		return nil, fmt.Errorf("JSON unmarshal: %w", err)
	}
	if a.Action == "" {
		return nil, fmt.Errorf("action field missing")
	}
	return &a, nil
}

// extractJSON finds the first JSON object in s.
// It checks for ```json fences first, then falls back to finding { }.
func extractJSON(s string) string {
	// Try fenced block first.
	if idx := strings.Index(s, "```json"); idx >= 0 {
		start := idx + len("```json")
		if end := strings.Index(s[start:], "```"); end >= 0 {
			return strings.TrimSpace(s[start : start+end])
		}
	}
	if idx := strings.Index(s, "```"); idx >= 0 {
		start := idx + 3
		if end := strings.Index(s[start:], "```"); end >= 0 {
			candidate := strings.TrimSpace(s[start : start+end])
			if strings.HasPrefix(candidate, "{") {
				return candidate
			}
		}
	}
	// Fall back to finding the first { ... } span.
	start := strings.Index(s, "{")
	if start < 0 {
		return ""
	}
	depth := 0
	for i := start; i < len(s); i++ {
		switch s[i] {
		case '{':
			depth++
		case '}':
			depth--
			if depth == 0 {
				return s[start : i+1]
			}
		}
	}
	return ""
}

// resolveSlug validates the proposed slug and handles collisions.
// On collision it appends "-2", "-3", etc. It also calls the LLM once for a
// retry if the initial slug is invalid.
func resolveSlug(
	ctx context.Context,
	proposed string,
	existingSlugs []string,
	cfg ModelConfig,
	sess *Session,
) (string, error) {
	slug := strings.ToLower(strings.TrimSpace(proposed))

	if !slugRe.MatchString(slug) {
		// Ask LLM for a corrected slug (one retry).
		retryMsg := []LLMMessage{
			{Role: "user", Content: fmt.Sprintf(
				`The slug "%s" is invalid. Please reply with ONLY a valid lowercase hyphenated slug (no spaces, start and end with alphanumeric).`,
				proposed,
			)},
		}
		raw, err := CallLLM(ctx, cfg, retryMsg)
		if err != nil {
			return "", fmt.Errorf("slug retry LLM call: %w", err)
		}
		slug = strings.ToLower(strings.TrimSpace(raw))
		// Strip any quotes or punctuation the model may have added.
		slug = strings.Trim(slug, `"' `)
		if !slugRe.MatchString(slug) {
			// Sanitise by replacing invalid chars.
			slug = sanitiseSlug(slug)
		}
	}

	if slug == "" {
		slug = "idea"
	}

	// Collision detection: append disambiguating suffix if needed.
	slugSet := make(map[string]bool, len(existingSlugs))
	for _, s := range existingSlugs {
		slugSet[s] = true
	}

	if !slugSet[slug] {
		return slug, nil
	}
	for i := 2; i <= 99; i++ {
		candidate := fmt.Sprintf("%s-%d", slug, i)
		if !slugSet[candidate] {
			return candidate, nil
		}
	}
	return slug + "-x", nil
}

// sanitiseSlug converts a string to a best-effort valid slug.
func sanitiseSlug(s string) string {
	re := regexp.MustCompile(`[^a-z0-9]+`)
	cleaned := re.ReplaceAllString(s, "-")
	cleaned = strings.Trim(cleaned, "-")
	if cleaned == "" {
		return "idea"
	}
	return cleaned
}

// filterLabels returns only the labels that exist in the allowed vocabulary.
// If vocabulary is empty, all labels are permitted (no vocabulary defined yet).
func filterLabels(proposed, allowed []string) []string {
	if len(allowed) == 0 {
		return proposed
	}
	set := make(map[string]bool, len(allowed))
	for _, l := range allowed {
		set[l] = true
	}
	var out []string
	for _, l := range proposed {
		if set[l] {
			out = append(out, l)
		}
	}
	return out
}
