// SPDX-License-Identifier: AGPL-3.0-or-later

// NOTE: Anthropic may change the wording of rate-limit messages without notice.
// The fallback_pause_minutes knob in the dispatcher config keeps us safe when
// the parser does not recognise the format. The WARN log of the raw text (in
// dispatcher.go) provides a paper trail so this parser can be extended. Add
// new patterns as branches in ParseResetTime below; cover each with a test row
// in parser_test.go.

package queue

import (
	"regexp"
	"strconv"
	"strings"
	"time"
)

// ParseResetTime extracts a reset time from a Claude rate-limit text payload.
// Returns (resetTime, true) on success or (time.Time{}, false) on unrecognised
// format.
//
// Supported formats (tried in order):
//  1. ISO 8601 / RFC3339 timestamp: "2026-05-12T20:00:00+10:00"
//  2. Relative: "retry after N seconds" / "retry after N minutes"
//  3. "resets HH:MMam/pm (TZ)" or "resets HHam/pm (TZ)" — with explicit TZ
//  4. "resets HH:MMam/pm" or "resets HHam/pm" — assumes server local TZ
func ParseResetTime(text string, now time.Time) (time.Time, bool) {
	if text == "" {
		return time.Time{}, false
	}

	// 1. ISO 8601 / RFC3339.
	if t, ok := parseRFC3339(text); ok {
		return t, true
	}

	// 2. "retry after N seconds/minutes".
	if t, ok := parseRetryAfter(text, now); ok {
		return t, true
	}

	// 3. "resets HH:MMam/pm (TZ)" with explicit timezone.
	// If the text contains the TZ-suffix form, handle it here exclusively — do
	// NOT fall through to the local-TZ parser even if the timezone name is
	// invalid, to avoid misinterpreting "resets 8pm (Bad/TZ)" as "resets 8pm"
	// in local time.
	if resetsWithTZRE.MatchString(text) {
		t, ok := parseResetsWithTZ(text, now)
		return t, ok
	}

	// 4. "resets HH:MMam/pm" — local TZ (only reached when no explicit TZ present).
	if t, ok := parseResetsLocal(text, now); ok {
		return t, true
	}

	return time.Time{}, false
}

// ---- pattern 1: RFC3339 ----

// Match ISO 8601 datetimes with a timezone suffix (Z or ±HH:MM), stopping
// before any trailing punctuation that cannot be part of an RFC3339 timestamp.
var rfc3339RE = regexp.MustCompile(`\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}(?:Z|[+-]\d{2}:\d{2})`)

func parseRFC3339(text string) (time.Time, bool) {
	m := rfc3339RE.FindString(text)
	if m == "" {
		return time.Time{}, false
	}
	t, err := time.Parse(time.RFC3339, m)
	if err != nil {
		// Try RFC3339Nano.
		t, err = time.Parse(time.RFC3339Nano, m)
	}
	if err != nil {
		return time.Time{}, false
	}
	return t, true
}

// ---- pattern 2: retry after N seconds/minutes ----

var retryAfterRE = regexp.MustCompile(`(?i)retry\s+after\s+(\d+)\s+(second|minute)s?`)

func parseRetryAfter(text string, now time.Time) (time.Time, bool) {
	m := retryAfterRE.FindStringSubmatch(text)
	if m == nil {
		return time.Time{}, false
	}
	n, err := strconv.Atoi(m[1])
	if err != nil || n < 0 {
		return time.Time{}, false
	}
	unit := strings.ToLower(m[2])
	var d time.Duration
	switch unit {
	case "second":
		d = time.Duration(n) * time.Second
	case "minute":
		d = time.Duration(n) * time.Minute
	default:
		return time.Time{}, false
	}
	return now.Add(d), true
}

// ---- pattern 3: "resets HH:MMam (TZ)" / "resets HHam (TZ)" ----

// Matches: resets 8:00pm (Australia/Brisbane)  resets 8pm (UTC)  etc.
var resetsWithTZRE = regexp.MustCompile(`(?i)resets?\s+(\d{1,2})(?::(\d{2}))?([ap]m)\s+\(([^)]+)\)`)

func parseResetsWithTZ(text string, now time.Time) (time.Time, bool) {
	m := resetsWithTZRE.FindStringSubmatch(text)
	if m == nil {
		return time.Time{}, false
	}
	hour, minute, ok := parseHourMinute(m[1], m[2], m[3])
	if !ok {
		return time.Time{}, false
	}
	loc, err := time.LoadLocation(m[4])
	if err != nil {
		return time.Time{}, false
	}
	return nextOccurrence(hour, minute, loc, now), true
}

// ---- pattern 4: "resets HH:MMam" / "resets HHam" — local TZ ----

var resetsLocalRE = regexp.MustCompile(`(?i)resets?\s+(\d{1,2})(?::(\d{2}))?([ap]m)`)

func parseResetsLocal(text string, now time.Time) (time.Time, bool) {
	m := resetsLocalRE.FindStringSubmatch(text)
	if m == nil {
		return time.Time{}, false
	}
	hour, minute, ok := parseHourMinute(m[1], m[2], m[3])
	if !ok {
		return time.Time{}, false
	}
	return nextOccurrence(hour, minute, now.Location(), now), true
}

// ---- helpers ----

// parseHourMinute converts "8", "00", "pm" → (20, 0, true).
func parseHourMinute(hourStr, minuteStr, ampm string) (hour, minute int, ok bool) {
	h, err := strconv.Atoi(hourStr)
	if err != nil {
		return 0, 0, false
	}
	// 12-hour clock validation: 1–12.
	if h < 1 || h > 12 {
		return 0, 0, false
	}
	m := 0
	if minuteStr != "" {
		m, err = strconv.Atoi(minuteStr)
		if err != nil || m < 0 || m > 59 {
			return 0, 0, false
		}
	}
	// Convert to 24-hour.
	switch strings.ToLower(ampm) {
	case "am":
		if h == 12 {
			h = 0
		}
	case "pm":
		if h != 12 {
			h += 12
		}
	default:
		return 0, 0, false
	}
	return h, m, true
}

// nextOccurrence returns the next wall-clock occurrence of hour:minute in loc
// after now. If today's occurrence is strictly in the future, it returns that;
// otherwise it returns tomorrow's.
func nextOccurrence(hour, minute int, loc *time.Location, now time.Time) time.Time {
	nowInLoc := now.In(loc)
	candidate := time.Date(nowInLoc.Year(), nowInLoc.Month(), nowInLoc.Day(), hour, minute, 0, 0, loc)
	if candidate.After(nowInLoc) {
		return candidate
	}
	return candidate.Add(24 * time.Hour)
}
