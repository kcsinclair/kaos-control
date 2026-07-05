// SPDX-License-Identifier: AGPL-3.0-or-later

package artifact

import (
	"errors"
	"regexp"
	"strings"
)

// Question is one parsed entry from a "## Open Questions" section.
type Question struct {
	Index  int    `json:"index"`
	Text   string `json:"text"`
	Answer string `json:"answer"`
}

// ErrIncompleteAnswers is returned by ApplyAnswers when complete is requested
// but at least one question has no answer.
var ErrIncompleteAnswers = errors.New("open questions: cannot complete, one or more questions has no answer")

// openQuestionsHeading is the exact heading text matched by HasOpenQuestions.
const openQuestionsHeading = "## Open Questions"

// resolvedQuestionsHeading is the heading ApplyAnswers renames to on completion.
const resolvedQuestionsHeading = "## Resolved Questions"

// topLevelItemRe matches a top-level "- " or "1. " list item marker at the
// start of a line, capturing the remainder of the line as question text.
var topLevelItemRe = regexp.MustCompile(`^(?:-|\d+\.)\s+(.*)$`)

// questionEntry is one raw top-level list item within an "## Open Questions"
// section: its text lines (verbatim, marker included on the first line) and
// any trailing blockquote answer already written under it.
type questionEntry struct {
	textLines []string
	answer    string
}

// locateOpenQuestionsSection returns the line index of the exact
// "## Open Questions" heading and the index of the line where the section
// ends (the next "## " heading, or len(lines) at EOF). Returns headingIdx=-1
// when the heading is absent, matching HasOpenQuestions semantics.
func locateOpenQuestionsSection(lines []string) (headingIdx, sectionEnd int) {
	headingIdx = -1
	for i, line := range lines {
		if line == openQuestionsHeading {
			headingIdx = i
			break
		}
	}
	if headingIdx == -1 {
		return -1, -1
	}
	sectionEnd = len(lines)
	for i := headingIdx + 1; i < len(lines); i++ {
		if strings.HasPrefix(lines[i], "## ") {
			sectionEnd = i
			break
		}
	}
	return headingIdx, sectionEnd
}

// splitQuestionEntries splits the lines of an "## Open Questions" section
// into one entry per top-level list item, separating out any trailing
// blockquote block as that entry's existing answer. Returns nil when the
// section contains no top-level list items.
func splitQuestionEntries(section []string) []questionEntry {
	var starts []int
	for i, line := range section {
		if topLevelItemRe.MatchString(line) {
			starts = append(starts, i)
		}
	}
	if len(starts) == 0 {
		return nil
	}

	entries := make([]questionEntry, 0, len(starts))
	for qi, start := range starts {
		end := len(section)
		if qi+1 < len(starts) {
			end = starts[qi+1]
		}
		entry := append([]string(nil), section[start:end]...)

		for len(entry) > 0 && strings.TrimSpace(entry[len(entry)-1]) == "" {
			entry = entry[:len(entry)-1]
		}

		// A trailing run of blockquote lines is this question's existing answer.
		answerStart := len(entry)
		for answerStart > 0 && strings.HasPrefix(strings.TrimLeft(entry[answerStart-1], " "), ">") {
			answerStart--
		}

		var answer string
		if answerStart < len(entry) {
			answerLines := make([]string, 0, len(entry)-answerStart)
			for _, l := range entry[answerStart:] {
				l = strings.TrimPrefix(strings.TrimLeft(l, " "), ">")
				answerLines = append(answerLines, strings.TrimPrefix(l, " "))
			}
			answer = strings.TrimSpace(strings.Join(answerLines, "\n"))
			entry = entry[:answerStart]
			for len(entry) > 0 && strings.TrimSpace(entry[len(entry)-1]) == "" {
				entry = entry[:len(entry)-1]
			}
		}

		entries = append(entries, questionEntry{textLines: entry, answer: answer})
	}

	return entries
}

// ParseOpenQuestions parses the "## Open Questions" section of body into an
// ordered list of questions. Each top-level list item under the heading
// (matched the same way as HasOpenQuestions: an exact "## Open Questions"
// line) is one question; a top-level list item's text runs until the next
// top-level item, a trailing blockquote (already-written answer), or the
// next "## " heading. Sub-items and prose belong to the preceding question.
//
// Returns (nil, false) when the section is absent or contains no top-level
// list items — this function never errors (NFR6 graceful parsing).
func ParseOpenQuestions(body string) ([]Question, bool) {
	lines := strings.Split(body, "\n")

	headingIdx, sectionEnd := locateOpenQuestionsSection(lines)
	if headingIdx == -1 {
		return nil, false
	}
	section := lines[headingIdx+1 : sectionEnd]

	entries := splitQuestionEntries(section)
	if entries == nil {
		return nil, false
	}

	questions := make([]Question, 0, len(entries))
	for qi, e := range entries {
		text := append([]string(nil), e.textLines...)
		text[0] = topLevelItemRe.FindStringSubmatch(text[0])[1]
		questions = append(questions, Question{
			Index:  qi,
			Text:   strings.TrimSpace(strings.Join(text, "\n")),
			Answer: e.answer,
		})
	}

	return questions, true
}

// ApplyAnswers builds a new body with the given answers written into the
// "## Open Questions" section, keyed by question index (as returned by
// ParseOpenQuestions). Questions not present in answers keep their existing
// answer. The answer for each question is written immediately after its
// text in the configured format (only "blockquote" is currently supported;
// any other value falls back to blockquote, matching
// OpenQuestionsConfig.EffectiveFormat). Frontmatter and all sections other
// than "## Open Questions" are left byte-for-byte unchanged.
//
// When complete is true, every question must end up with a non-empty answer
// (existing or supplied); on success the heading is renamed to
// "## Resolved Questions" in the same returned body. When complete is true
// and any answer is empty, ApplyAnswers returns ErrIncompleteAnswers and
// does not modify the body.
//
// Applying identical inputs twice produces byte-identical output.
func ApplyAnswers(body string, answers map[int]string, format string, complete bool) (string, error) {
	lines := strings.Split(body, "\n")

	headingIdx, sectionEnd := locateOpenQuestionsSection(lines)
	if headingIdx == -1 {
		return body, nil
	}
	section := lines[headingIdx+1 : sectionEnd]

	entries := splitQuestionEntries(section)
	if entries == nil {
		return body, nil
	}

	finalAnswers := make([]string, len(entries))
	for i, e := range entries {
		finalAnswers[i] = e.answer
	}
	for idx, ans := range answers {
		if idx >= 0 && idx < len(finalAnswers) {
			finalAnswers[idx] = ans
		}
	}

	if complete {
		for _, a := range finalAnswers {
			if strings.TrimSpace(a) == "" {
				return "", ErrIncompleteAnswers
			}
		}
	}

	var rebuilt []string
	for i, e := range entries {
		rebuilt = append(rebuilt, e.textLines...)
		if ans := strings.TrimSpace(finalAnswers[i]); ans != "" {
			rebuilt = append(rebuilt, "")
			for _, al := range strings.Split(ans, "\n") {
				rebuilt = append(rebuilt, "> "+al)
			}
		}
		if i != len(entries)-1 {
			rebuilt = append(rebuilt, "")
		}
	}

	heading := openQuestionsHeading
	if complete {
		heading = resolvedQuestionsHeading
	}

	out := make([]string, 0, headingIdx+2+len(rebuilt)+1+(len(lines)-sectionEnd))
	out = append(out, lines[:headingIdx]...)
	out = append(out, heading, "")
	out = append(out, rebuilt...)
	out = append(out, "")
	out = append(out, lines[sectionEnd:]...)

	return strings.Join(out, "\n"), nil
}
