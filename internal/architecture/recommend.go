// SPDX-License-Identifier: AGPL-3.0-or-later

package architecture

import (
	"sort"

	"github.com/kaos-control/kaos-control/internal/config"
)

// Answer is one questionnaire response: the question ID and the value of the
// option the user picked (or a free-form value for questions with no fixed
// options).
type Answer struct {
	QuestionID string `json:"question_id"`
	Value      string `json:"value"`
}

// Recommendation is one scored architecture candidate.
type Recommendation struct {
	Item  CatalogItem `json:"item"`
	Score int         `json:"score"`
	Why   []string    `json:"why"`
}

// maxRecommendations caps the number of candidates Recommend returns (FR-9).
const maxRecommendations = 3

// labelContribution is one answer's contribution to the scoring pass: a
// catalog label, whether it acts as a hard constraint, and a human-readable
// reason tying it back to the question/answer that produced it.
type labelContribution struct {
	label string
	hard  bool
	why   string
}

// Recommend scores arches against answers (FR-8–FR-12): hard-constraint
// labels filter the candidate set (relaxing the weakest one at a time on a
// zero-match result, OQ-2), soft-signal labels score the survivors by label
// overlap, and the result is deterministic — identical inputs always yield
// an identical ranking (FR-12). wizardCfg supplies both the question set (for
// answer→label resolution) and DefaultArchitecture (the low-regret bias
// target, FR-11).
func Recommend(arches []CatalogItem, wizardCfg config.ArchitectureWizardConfig, answers []Answer) (recs []Recommendation, droppedConstraints []string, err error) {
	byID := make(map[string]config.WizardQuestion, len(wizardCfg.Questions))
	for _, q := range wizardCfg.Questions {
		byID[q.ID] = q
	}

	var hard, soft []labelContribution
	for _, ans := range answers {
		q, ok := byID[ans.QuestionID]
		if !ok || q.Kind == "language" {
			continue
		}
		opt, labels := resolveOption(q, ans.Value)
		isHard := q.Kind == "hard" || (opt != nil && opt.Hard)
		for _, label := range labels {
			c := labelContribution{label: label, hard: isHard, why: contributionWhy(q, opt, label)}
			if isHard {
				hard = append(hard, c)
			} else {
				soft = append(soft, c)
			}
		}
	}

	hardLabels := dedupContributions(hard)
	survivors, dropped := filterByHardConstraints(arches, hardLabels)

	softLabels := dedupContributions(soft)
	recs = scoreSurvivors(survivors, hardLabels, softLabels)

	defaultSlug := wizardCfg.DefaultArchitecture
	if defaultSlug == "" {
		defaultSlug = "modular-monolith"
	}
	applyDefaultBias(recs, defaultSlug)

	if len(recs) > maxRecommendations {
		recs = recs[:maxRecommendations]
	}
	return recs, dropped, nil
}

// resolveOption finds the option matching value on question q (nil if the
// question carries no fixed options or the value doesn't match one), and
// returns the labels it contributes: the matched option's Labels, or
// question-level nothing if no options are configured for a free-form
// question.
func resolveOption(q config.WizardQuestion, value string) (*config.WizardOption, []string) {
	for i := range q.Options {
		if q.Options[i].Value == value {
			return &q.Options[i], q.Options[i].Labels
		}
	}
	return nil, nil
}

func contributionWhy(q config.WizardQuestion, opt *config.WizardOption, label string) string {
	answerLabel := label
	if opt != nil && opt.Label != "" {
		answerLabel = opt.Label
	}
	return q.Prompt + " → " + answerLabel
}

// dedupContributions collapses contributions to one entry per distinct
// label, in deterministic (label-name) order, keeping the first why seen for
// that label.
func dedupContributions(cs []labelContribution) []labelContribution {
	seen := make(map[string]labelContribution, len(cs))
	for _, c := range cs {
		if _, ok := seen[c.label]; !ok {
			seen[c.label] = c
		}
	}
	out := make([]labelContribution, 0, len(seen))
	for _, c := range seen {
		out = append(out, c)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].label < out[j].label })
	return out
}

// filterByHardConstraints returns the arches satisfying every hard label. If
// that set is empty, it relaxes constraints one at a time — weakest first,
// defined as the label with the fewest matches across the full catalog, tied
// broken alphabetically — recording each dropped label, until at least one
// candidate survives or no constraints remain (OQ-2).
func filterByHardConstraints(arches []CatalogItem, hardLabels []labelContribution) (survivors []CatalogItem, dropped []string) {
	remaining := append([]labelContribution(nil), hardLabels...)

	for {
		survivors = matchAll(arches, remaining)
		if len(survivors) > 0 || len(remaining) == 0 {
			return survivors, dropped
		}

		weakestIdx := 0
		weakestCount := matchCount(arches, remaining[0].label)
		for i := 1; i < len(remaining); i++ {
			c := matchCount(arches, remaining[i].label)
			if c < weakestCount || (c == weakestCount && remaining[i].label < remaining[weakestIdx].label) {
				weakestIdx, weakestCount = i, c
			}
		}

		dropped = append(dropped, remaining[weakestIdx].label)
		remaining = append(remaining[:weakestIdx], remaining[weakestIdx+1:]...)
	}
}

// matchAll returns the arches carrying every label in required.
func matchAll(arches []CatalogItem, required []labelContribution) []CatalogItem {
	var out []CatalogItem
	for _, a := range arches {
		if hasAllLabels(a.Labels, required) {
			out = append(out, a)
		}
	}
	return out
}

func hasAllLabels(itemLabels []string, required []labelContribution) bool {
	set := make(map[string]bool, len(itemLabels))
	for _, l := range itemLabels {
		set[l] = true
	}
	for _, r := range required {
		if !set[r.label] {
			return false
		}
	}
	return true
}

// matchCount returns how many arches in the full catalog carry label.
func matchCount(arches []CatalogItem, label string) int {
	n := 0
	for _, a := range arches {
		for _, l := range a.Labels {
			if l == label {
				n++
				break
			}
		}
	}
	return n
}

// scoreSurvivors scores each survivor by soft-label overlap and builds its
// Why from the hard and soft contributions it actually satisfies, then
// stable-sorts by (score desc, DefaultArchitecture-first is applied
// separately, slug asc).
func scoreSurvivors(survivors []CatalogItem, hardLabels, softLabels []labelContribution) []Recommendation {
	recs := make([]Recommendation, 0, len(survivors))
	for _, item := range survivors {
		itemLabels := make(map[string]bool, len(item.Labels))
		for _, l := range item.Labels {
			itemLabels[l] = true
		}

		var why []string
		for _, c := range hardLabels {
			if itemLabels[c.label] {
				why = append(why, c.why)
			}
		}
		score := 0
		for _, c := range softLabels {
			if itemLabels[c.label] {
				score++
				why = append(why, c.why)
			}
		}
		recs = append(recs, Recommendation{Item: item, Score: score, Why: why})
	}

	sort.SliceStable(recs, func(i, j int) bool {
		if recs[i].Score != recs[j].Score {
			return recs[i].Score > recs[j].Score
		}
		return recs[i].Item.Slug < recs[j].Item.Slug
	})
	return recs
}

// applyDefaultBias implements FR-11: when signals are too weak or ambiguous
// to distinguish candidates (the top score is zero), the low-regret default
// architecture (defaultSlug) is promoted to the front with a documented bias
// Why, if it is among the survivors.
func applyDefaultBias(recs []Recommendation, defaultSlug string) {
	if len(recs) == 0 || recs[0].Score > 0 {
		return
	}
	for i, r := range recs {
		if r.Item.Slug != defaultSlug {
			continue
		}
		if i != 0 {
			recs[0], recs[i] = recs[i], recs[0]
		}
		recs[0].Why = []string{"low-regret default — signals were weak"}
		return
	}
}

// RankStacks restricts stacks to those related_to the chosen architecture,
// then stable-sorts so stacks whose labels include languageAnswer come first
// (FR-6, FR-10). Returns an empty slice, never nil-panics, if chosen has no
// related_to stacks or a catalog entry is malformed.
func RankStacks(chosen CatalogItem, stacks []CatalogItem, languageAnswer string) []CatalogItem {
	related := make(map[string]bool, len(chosen.RelatedTo))
	for _, r := range chosen.RelatedTo {
		related[r] = true
	}

	var compatible []CatalogItem
	for _, s := range stacks {
		if related[s.Path] {
			compatible = append(compatible, s)
		}
	}

	sort.SliceStable(compatible, func(i, j int) bool {
		iMatch := containsLabel(compatible[i].Labels, languageAnswer)
		jMatch := containsLabel(compatible[j].Labels, languageAnswer)
		if iMatch != jMatch {
			return iMatch
		}
		return false // preserve incoming (slug-sorted) order within a match tier
	})
	return compatible
}

func containsLabel(labels []string, label string) bool {
	if label == "" {
		return false
	}
	for _, l := range labels {
		if l == label {
			return true
		}
	}
	return false
}
