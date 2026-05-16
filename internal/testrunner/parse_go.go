// SPDX-License-Identifier: AGPL-3.0-or-later

package testrunner

import (
	"bufio"
	"encoding/json"
	"io"
	"regexp"
	"strconv"
	"strings"
)

// goTestEvent is one line from `go test -json` output.
type goTestEvent struct {
	Action  string  `json:"Action"`
	Package string  `json:"Package"`
	Test    string  `json:"Test"`
	Output  string  `json:"Output"`
	Elapsed float64 `json:"Elapsed"`
}

type goTestAccum struct {
	outputs []string
}

// ParseGoJSON reads the newline-delimited JSON stream from `go test -json`.
// If the input is not valid JSON (e.g. a compilation error produced plain text),
// the returned SuiteResult has RawError set and no Failures.
func ParseGoJSON(r io.Reader) (*SuiteResult, error) {
	result := &SuiteResult{Suite: "go"}

	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 1<<20), 1<<20) // 1 MiB per line max
	var lines []string
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}

	// Detect whether the output is a valid JSON stream by checking the first
	// non-empty line. Non-JSON means a compilation failure or build error.
	validJSON := false
	for _, l := range lines {
		if strings.TrimSpace(l) == "" {
			continue
		}
		var probe map[string]any
		if json.Unmarshal([]byte(l), &probe) == nil {
			validJSON = true
		}
		break
	}

	if !validJSON && len(lines) > 0 {
		result.RawError = strings.Join(lines, "\n")
		return result, nil
	}

	// accumulate per-test output so we can extract file:line:msg on failure.
	accum := make(map[string]*goTestAccum)

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		var evt goTestEvent
		if err := json.Unmarshal([]byte(line), &evt); err != nil {
			continue
		}

		key := evt.Package + "::" + evt.Test

		switch evt.Action {
		case "run":
			if evt.Test != "" {
				accum[key] = &goTestAccum{}
				result.Total++
			}
		case "output":
			if evt.Test != "" {
				if a, ok := accum[key]; ok {
					a.outputs = append(a.outputs, evt.Output)
				}
			}
		case "pass":
			if evt.Test != "" {
				result.Passed++
				result.Elapsed += evt.Elapsed
			}
		case "fail":
			if evt.Test != "" {
				result.Failed++
				result.Elapsed += evt.Elapsed

				var output string
				if a, ok := accum[key]; ok {
					output = strings.Join(a.outputs, "")
				}

				file, lineN, errMsg := parseGoOutputFileLine(output)

				result.Failures = append(result.Failures, TestFailure{
					Suite:    "go",
					Package:  evt.Package,
					TestName: evt.Test,
					File:     file,
					Line:     lineN,
					ErrorMsg: errMsg,
					Output:   output,
					Elapsed:  evt.Elapsed,
				})
			}
		case "skip":
			if evt.Test != "" {
				result.Skipped++
			}
		}
	}

	return result, nil
}

// goOutputFileLineRe matches the standard Go test failure line format:
//
//	    file_test.go:42: error message here
var goOutputFileLineRe = regexp.MustCompile(`(?m)^\s+(\S+\.go):(\d+): (.+)$`)

// parseGoOutputFileLine extracts the primary file, line number, and error
// message from accumulated Go test output.
func parseGoOutputFileLine(output string) (file string, line int, msg string) {
	m := goOutputFileLineRe.FindStringSubmatch(output)
	if m == nil {
		return "", 0, ""
	}
	n, _ := strconv.Atoi(m[2])
	return m[1], n, strings.TrimSpace(m[3])
}
