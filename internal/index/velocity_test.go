package index

// Milestone 1 — Unit tests for CompletionVelocity days parameter handling.
// Milestone 2 — Unit tests for zero-filled period coverage.
//
// These tests exercise CompletionVelocity and the internal velocityPeriods
// helper directly (package-internal access). They use an empty in-memory
// SQLite index so no events ever match, allowing clean isolation of the
// parameter-handling logic.

import (
	"testing"
	"time"
)

// openVelocityTestIndex opens a minimal temp-dir SQLite index with no stages
// configured. The index is empty (no artifacts, no events).
func openVelocityTestIndex(t *testing.T) *Index {
	t.Helper()
	dir := t.TempDir()
	idx, err := Open(dir+"/velocity_test.db", dir, nil)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { idx.Close() })
	return idx
}

// TestVelocityDays_ZeroFallsToDefault verifies that days=0 is treated as the
// default (90), producing the same bucket count as days=90.
func TestVelocityDays_ZeroFallsToDefault(t *testing.T) {
	idx := openVelocityTestIndex(t)

	b0, err := idx.CompletionVelocity("daily", 0, nil)
	if err != nil {
		t.Fatalf("days=0: %v", err)
	}
	b90, err := idx.CompletionVelocity("daily", 90, nil)
	if err != nil {
		t.Fatalf("days=90: %v", err)
	}

	// Allow ±1 for a possible midnight crossing between calls.
	diff := len(b0) - len(b90)
	if diff < -1 || diff > 1 {
		t.Errorf("days=0 produced %d buckets, days=90 produced %d buckets; want same (±1)", len(b0), len(b90))
	}
}

// TestVelocityDays_NegativeFallsToDefault verifies that days=-1 is treated as
// the default (90), producing the same bucket count as days=90.
func TestVelocityDays_NegativeFallsToDefault(t *testing.T) {
	idx := openVelocityTestIndex(t)

	bNeg, err := idx.CompletionVelocity("daily", -1, nil)
	if err != nil {
		t.Fatalf("days=-1: %v", err)
	}
	b90, err := idx.CompletionVelocity("daily", 90, nil)
	if err != nil {
		t.Fatalf("days=90: %v", err)
	}

	diff := len(bNeg) - len(b90)
	if diff < -1 || diff > 1 {
		t.Errorf("days=-1 produced %d buckets, days=90 produced %d buckets; want same (±1)", len(bNeg), len(b90))
	}
}

// TestVelocityDays_ExceedsMaxClamped verifies that days=366 and days=400 are
// both clamped to 365, producing the same bucket count as days=365.
func TestVelocityDays_ExceedsMaxClamped(t *testing.T) {
	idx := openVelocityTestIndex(t)

	b365, err := idx.CompletionVelocity("daily", 365, nil)
	if err != nil {
		t.Fatalf("days=365: %v", err)
	}

	for _, over := range []int{366, 400} {
		bOver, err := idx.CompletionVelocity("daily", over, nil)
		if err != nil {
			t.Fatalf("days=%d: %v", over, err)
		}
		diff := len(bOver) - len(b365)
		if diff < -1 || diff > 1 {
			t.Errorf("days=%d produced %d buckets, days=365 produced %d buckets; want same (±1)", over, len(bOver), len(b365))
		}
	}
}

// TestVelocityDays_ExplicitFourteen verifies that days=14 returns exactly 15
// daily buckets (14 prior days + today, both endpoints inclusive).
func TestVelocityDays_ExplicitFourteen(t *testing.T) {
	idx := openVelocityTestIndex(t)

	buckets, err := idx.CompletionVelocity("daily", 14, nil)
	if err != nil {
		t.Fatalf("CompletionVelocity: %v", err)
	}

	// Daily: both `since` and `now` are midnight-truncated, so for days=14 the
	// window spans exactly 15 days (inclusive on both ends). Allow ±1 for a
	// midnight crossing between the call and our calculation.
	if len(buckets) < 14 || len(buckets) > 16 {
		t.Errorf("days=14 daily: want 15 buckets (±1), got %d", len(buckets))
	}
}

// TestVelocityPeriods_DailyFixed verifies period count and ordering for a
// known date range using the internal velocityPeriods helper.
func TestVelocityPeriods_DailyFixed(t *testing.T) {
	loc := time.UTC
	since := time.Date(2026, 1, 1, 0, 0, 0, 0, loc)
	now := time.Date(2026, 1, 7, 0, 0, 0, 0, loc)
	periods := velocityPeriods("daily", since, now)

	if len(periods) != 7 {
		t.Fatalf("want 7 daily periods, got %d: %v", len(periods), periods)
	}
	if periods[0] != "2026-01-01" {
		t.Errorf("periods[0] = %q, want 2026-01-01", periods[0])
	}
	if periods[6] != "2026-01-07" {
		t.Errorf("periods[6] = %q, want 2026-01-07", periods[6])
	}
}

// TestVelocityPeriods_WeeklyFixed verifies weekly period count for a 28-day
// window where both endpoints are Mondays.
func TestVelocityPeriods_WeeklyFixed(t *testing.T) {
	loc := time.UTC
	// April 6 2026 = Monday (W15). May 4 2026 = Monday (W19). 28 days apart.
	since := time.Date(2026, 4, 6, 0, 0, 0, 0, loc)
	now := time.Date(2026, 5, 4, 0, 0, 0, 0, loc)
	periods := velocityPeriods("weekly", since, now)

	// W15, W16, W17, W18, W19 = 5 periods.
	if len(periods) != 5 {
		t.Fatalf("want 5 weekly periods, got %d: %v", len(periods), periods)
	}
	if periods[0] != "2026-W15" {
		t.Errorf("periods[0] = %q, want 2026-W15", periods[0])
	}
	if periods[4] != "2026-W19" {
		t.Errorf("periods[4] = %q, want 2026-W19", periods[4])
	}
}

// TestVelocityPeriods_MonthlyFixed verifies monthly period count and labels
// for a known 3-month range.
func TestVelocityPeriods_MonthlyFixed(t *testing.T) {
	loc := time.UTC
	since := time.Date(2026, 1, 15, 0, 0, 0, 0, loc) // mid-January
	now := time.Date(2026, 3, 20, 0, 0, 0, 0, loc)   // mid-March
	periods := velocityPeriods("monthly", since, now)

	// First of Jan → first of Mar: Jan, Feb, Mar = 3 periods.
	if len(periods) != 3 {
		t.Fatalf("want 3 monthly periods, got %d: %v", len(periods), periods)
	}
	if periods[0] != "2026-01" {
		t.Errorf("periods[0] = %q, want 2026-01", periods[0])
	}
	if periods[2] != "2026-03" {
		t.Errorf("periods[2] = %q, want 2026-03", periods[2])
	}
}

// ----- Milestone 2: zero-fill coverage for an empty project -----

// assertAllZero fails the test if any bucket in bs has a non-zero count.
func assertAllZero(t *testing.T, bs []VelocityBucket) {
	t.Helper()
	for _, b := range bs {
		if b.Count != 0 {
			t.Errorf("period %q: want count=0, got %d", b.Period, b.Count)
		}
	}
}

// TestVelocityZeroFill_Daily verifies that an empty project returns a
// contiguous, zero-valued daily series covering the requested window.
func TestVelocityZeroFill_Daily(t *testing.T) {
	idx := openVelocityTestIndex(t)

	const days = 7
	buckets, err := idx.CompletionVelocity("daily", days, nil)
	if err != nil {
		t.Fatalf("CompletionVelocity: %v", err)
	}
	// days=7 produces 8 periods (0..7 inclusive, both midnight-truncated).
	// Allow ±1 for a midnight crossing between the call and our assertion.
	if len(buckets) < days || len(buckets) > days+2 {
		t.Errorf("want %d buckets (±1), got %d", days+1, len(buckets))
	}
	assertAllZero(t, buckets)
}

// TestVelocityZeroFill_Weekly verifies that an empty project returns a
// contiguous, zero-valued weekly series. days=28 (a multiple of 7) always
// yields exactly 5 ISO-week buckets regardless of the current day of the week.
func TestVelocityZeroFill_Weekly(t *testing.T) {
	idx := openVelocityTestIndex(t)

	const days = 28
	buckets, err := idx.CompletionVelocity("weekly", days, nil)
	if err != nil {
		t.Fatalf("CompletionVelocity: %v", err)
	}
	// 28 is 4 full weeks. Because since = now-28 always shares the same weekday
	// as now, isoWeekMonday(since) is exactly 28 days before isoWeekMonday(now),
	// producing 5 Monday-anchored periods (offsets 0, 7, 14, 21, 28).
	if len(buckets) != 5 {
		t.Errorf("days=28 weekly: want 5 buckets, got %d", len(buckets))
	}
	assertAllZero(t, buckets)
}

// TestVelocityZeroFill_Monthly verifies that an empty project returns a
// contiguous, zero-valued monthly series for a 90-day window.
func TestVelocityZeroFill_Monthly(t *testing.T) {
	idx := openVelocityTestIndex(t)

	const days = 90
	buckets, err := idx.CompletionVelocity("monthly", days, nil)
	if err != nil {
		t.Fatalf("CompletionVelocity: %v", err)
	}

	// Determine expected bucket count using the same logic as CompletionVelocity.
	now := time.Now()
	since := now.AddDate(0, 0, -days)
	expected := velocityPeriods("monthly", since, now)

	// Allow ±1 because time.Now() is called twice.
	diff := len(buckets) - len(expected)
	if diff < -1 || diff > 1 {
		t.Errorf("days=90 monthly: want ~%d buckets (±1), got %d", len(expected), len(buckets))
	}
	if len(buckets) < 2 {
		t.Errorf("days=90 monthly: want at least 2 buckets, got %d", len(buckets))
	}
	assertAllZero(t, buckets)
}

// TestVelocityZeroFill_NoBuckets verifies that days=0 (defaulted to 90)
// still returns a non-nil, non-empty zero-filled slice.
func TestVelocityZeroFill_DefaultedDays(t *testing.T) {
	idx := openVelocityTestIndex(t)

	buckets, err := idx.CompletionVelocity("daily", 0, nil)
	if err != nil {
		t.Fatalf("CompletionVelocity: %v", err)
	}
	if buckets == nil {
		t.Fatal("want non-nil slice, got nil")
	}
	if len(buckets) == 0 {
		t.Fatal("want non-empty slice, got empty")
	}
	assertAllZero(t, buckets)
}
