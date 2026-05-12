// SPDX-License-Identifier: AGPL-3.0-or-later

package queue

import (
	"testing"
	"time"
)

// brisbaneLoc is used throughout tests to avoid repeated LoadLocation calls.
var brisbaneLoc, _ = time.LoadLocation("Australia/Brisbane")

// fixedNow is a reference instant used as "now" in parser tests.
// 2026-05-12 15:00:00 Brisbane (UTC+10) = 2026-05-12 05:00:00 UTC.
var fixedNow = time.Date(2026, 5, 12, 15, 0, 0, 0, brisbaneLoc)

// parseCase exercises one input and asserts the result is correct.
type parseCase struct {
	name string
	text string
	// When wantOk is true, wantTime must equal the result (truncated to second).
	wantOk   bool
	wantTime time.Time
}

func runCases(t *testing.T, cases []parseCase) {
	t.Helper()
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, ok := ParseResetTime(tc.text, fixedNow)
			if ok != tc.wantOk {
				t.Fatalf("ok=%v, want %v (text=%q)", ok, tc.wantOk, tc.text)
			}
			if !tc.wantOk {
				return
			}
			// Compare truncated to second to avoid nanosecond noise.
			if !got.Truncate(time.Second).Equal(tc.wantTime.Truncate(time.Second)) {
				t.Errorf("got %v, want %v", got.Truncate(time.Second), tc.wantTime.Truncate(time.Second))
			}
		})
	}
}

func TestParseRFC3339(t *testing.T) {
	runCases(t, []parseCase{
		{
			name:     "bare timestamp",
			text:     "Rate limit exceeded. Reset at 2026-05-12T20:00:00+10:00.",
			wantOk:   true,
			wantTime: time.Date(2026, 5, 12, 20, 0, 0, 0, time.FixedZone("", 10*3600)),
		},
		{
			name:     "UTC timestamp",
			text:     "retry after 2026-05-12T10:00:00Z",
			wantOk:   true,
			wantTime: time.Date(2026, 5, 12, 10, 0, 0, 0, time.UTC),
		},
	})
}

func TestParseRetryAfter(t *testing.T) {
	runCases(t, []parseCase{
		{
			name:     "seconds",
			text:     "Rate limited. Retry after 120 seconds.",
			wantOk:   true,
			wantTime: fixedNow.Add(120 * time.Second),
		},
		{
			name:     "minutes",
			text:     "Too many requests. Retry after 5 minutes.",
			wantOk:   true,
			wantTime: fixedNow.Add(5 * time.Minute),
		},
		{
			name:     "case insensitive",
			text:     "RETRY AFTER 30 SECONDS",
			wantOk:   true,
			wantTime: fixedNow.Add(30 * time.Second),
		},
		{
			name:     "1 second singular",
			text:     "Retry after 1 second",
			wantOk:   true,
			wantTime: fixedNow.Add(time.Second),
		},
	})
}

func TestParseResetsWithTZ(t *testing.T) {
	// fixedNow = 15:00 Brisbane. "resets 8pm Brisbane" = today 20:00 Brisbane (still future).
	brisbane8pm := time.Date(2026, 5, 12, 20, 0, 0, 0, brisbaneLoc)
	// "resets 8am Brisbane" = today 08:00 Brisbane (past → tomorrow).
	brisbane8amTomorrow := time.Date(2026, 5, 13, 8, 0, 0, 0, brisbaneLoc)

	utcLoc := time.UTC
	utc6pm := time.Date(2026, 5, 12, 18, 0, 0, 0, utcLoc)

	runCases(t, []parseCase{
		{
			name:     "8pm Brisbane today",
			text:     "Claude rate limit hit. Resets 8pm (Australia/Brisbane).",
			wantOk:   true,
			wantTime: brisbane8pm,
		},
		{
			name:     "8:00pm Brisbane today",
			text:     "Resets 8:00pm (Australia/Brisbane)",
			wantOk:   true,
			wantTime: brisbane8pm,
		},
		{
			name:     "8am Brisbane past → tomorrow",
			text:     "resets 8am (Australia/Brisbane)",
			wantOk:   true,
			wantTime: brisbane8amTomorrow,
		},
		{
			name:     "6pm UTC",
			text:     "Rate limit. resets 6pm (UTC)",
			wantOk:   true,
			wantTime: utc6pm,
		},
	})
}

func TestParseResetsLocal(t *testing.T) {
	// Local TZ of fixedNow is Brisbane.
	brisbane8pm := time.Date(2026, 5, 12, 20, 0, 0, 0, brisbaneLoc)

	runCases(t, []parseCase{
		{
			name:     "8pm local",
			text:     "resets 8pm",
			wantOk:   true,
			wantTime: brisbane8pm,
		},
		{
			name:     "8:30pm local",
			text:     "Resets 8:30pm",
			wantOk:   true,
			wantTime: time.Date(2026, 5, 12, 20, 30, 0, 0, brisbaneLoc),
		},
	})
}

func TestParseMalformed(t *testing.T) {
	runCases(t, []parseCase{
		{name: "empty", text: "", wantOk: false},
		{name: "resets soon", text: "resets soon", wantOk: false},
		{name: "invalid hour 25pm", text: "resets 25pm (Australia/Brisbane)", wantOk: false},
		{name: "invalid timezone", text: "resets 8pm (TZ/Made-Up)", wantOk: false},
		{name: "no time at all", text: "rate limit exceeded please wait", wantOk: false},
	})
}

// TestNextOccurrenceAlwaysAfterNow is a property test: for any valid hour in a
// known TZ, the result of nextOccurrence is strictly after now.
func TestNextOccurrenceAlwaysAfterNow(t *testing.T) {
	locs := []string{"Australia/Brisbane", "UTC", "America/New_York"}
	for _, locName := range locs {
		loc, _ := time.LoadLocation(locName)
		for hour := 0; hour < 24; hour++ {
			got := nextOccurrence(hour, 0, loc, fixedNow)
			if !got.After(fixedNow) {
				t.Errorf("loc=%s hour=%d: %v is not after now %v", locName, hour, got, fixedNow)
			}
		}
	}
}
