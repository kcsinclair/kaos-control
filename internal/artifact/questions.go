// SPDX-License-Identifier: AGPL-3.0-or-later

package artifact

import (
	"regexp"
	"strings"
)

// Question is one parsed entry from a "## Open Questions" section.
type Question struct {
	Index  int    `json:"index"`
	Text   string `json:"text"`
	Answer string `json:"answer"`
}

// topLevelItemRe matches a top-level "- " or "1. " list item marker at the
// start of a line, capturing the remainder of the line as question text.
var topLevelItemRe = regexp.MustCompile(`^(?:-|\d+\.)\s+(.*)$`)

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

	headingIdx := -1
	for i, line := range lines {
		if line == "## Open Questions" {
			headingIdx = i
			break
		}
	}
	if headingIdx == -1 {
		return nil, false
	}

	var section []string
	for _, line := range lines[headingIdx+1:] {
		if strings.HasPrefix(line, "## ") {
			break
		}
		section = append(section, line)
	}

	var starts []int
	for i, line := range section {
		if topLevelItemRe.MatchString(line) {
			starts = append(starts, i)
		}
	}
	if len(starts) == 0 {
		return nil, false
	}

	questions := make([]Question, 0, len(starts))
	for qi, start := range starts {
		end := len(section)
		if qi+1 < len(starts) {
			end = starts[qi+1]
		}
		entry := append([]string(nil), section[start:end]...)
		entry[0] = topLevelItemRe.FindStringSubmatch(entry[0])[1]

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

		questions = append(questions, Question{
			Index:  qi,
			Text:   strings.TrimSpace(strings.Join(entry, "\n")),
			Answer: answer,
		})
	}

	return questions, true
}
