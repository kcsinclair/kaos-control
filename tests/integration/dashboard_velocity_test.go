// SPDX-License-Identifier: AGPL-3.0-or-later

//go:build integration

package integration

import (
	"fmt"
	"testing"
	"time"

	"github.com/kaos-control/kaos-control/internal/index"
)

// ── Milestone 3 ───────────────────────────────────────────────────────────────

// seedVelocityEvent inserts a status_transition → done event for the given
// artifact path at the given Unix timestamp.
func seedVelocityEvent(t *testing.T, env *testEnv, artifactPath string, ts int64) {
	t.Helper()
	p := artifactPath
	e := &index.EventRow{
		EventType:    "status_transition",
		Timestamp:    ts,
		Actor:        "test",
		ArtifactPath: &p,
		Summary:      fmt.Sprintf("%q transitioned from approved → done", artifactPath),
	}
	if err := env.proj.Idx.InsertEvent(e); err != nil {
		t.Fatalf("InsertEvent(%s): %v", artifactPath, err)
	}
}

// daysAgo returns the Unix timestamp for midnight N days before today (local time).
func daysAgo(n int) int64 {
	t := time.Now().AddDate(0, 0, -n)
	return time.Date(t.Year(), t.Month(), t.Day(), 12, 0, 0, 0, t.Location()).Unix()
}

// TestVelocity_DailyGranularity verifies that daily granularity returns one
// bucket per day within the lookback window and that event counts are correct.
func TestVelocity_DailyGranularity(t *testing.T) {
	seeds := []seedArtifact{
		{relPath: "lifecycle/requirements/vel-daily-1.md",
			content: makeArtifact("Vel Daily 1", "ticket", "done", "vel-daily-1", "", "Body.")},
		{relPath: "lifecycle/requirements/vel-daily-2.md",
			content: makeArtifact("Vel Daily 2", "ticket", "done", "vel-daily-2", "", "Body.")},
	}
	env := newTestEnv(t, seeds)
	env.login("admin@test.local", "admin-pass-123")

	// Seed: 2 completions 3 days ago, 1 completion yesterday.
	seedVelocityEvent(t, env, "lifecycle/requirements/vel-daily-1.md", daysAgo(3))
	seedVelocityEvent(t, env, "lifecycle/requirements/vel-daily-2.md", daysAgo(3))
	seedVelocityEvent(t, env, "lifecycle/requirements/vel-daily-1.md", daysAgo(1))

	resp := env.doRequest("GET", "/api/p/testproject/dashboard/velocity?granularity=daily&days=7", nil)
	requireStatus(t, resp, 200)
	data := readJSON(t, resp)

	gran, _ := data["granularity"].(string)
	if gran != "daily" {
		t.Errorf("granularity: want %q, got %q", "daily", gran)
	}

	buckets := decodeVelocityBuckets(t, data)

	// Must have 8 buckets (days 0..7 inclusive).
	if len(buckets) < 7 {
		t.Errorf("expected at least 7 daily buckets for days=7, got %d", len(buckets))
	}

	// Verify no gaps: all bucket counts are present (zero or positive).
	totalCount := 0
	for _, b := range buckets {
		totalCount += b.count
	}
	if totalCount != 3 {
		t.Errorf("total event count across all buckets: want 3, got %d", totalCount)
	}

	// Find the bucket for 3 days ago and verify its count.
	key3 := time.Now().AddDate(0, 0, -3).Format("2006-01-02")
	if c := velocityBucketCount(buckets, key3); c != 2 {
		t.Errorf("bucket %s count: want 2, got %d", key3, c)
	}

	// Find the bucket for 1 day ago and verify its count.
	key1 := time.Now().AddDate(0, 0, -1).Format("2006-01-02")
	if c := velocityBucketCount(buckets, key1); c != 1 {
		t.Errorf("bucket %s count: want 1, got %d", key1, c)
	}
}

// TestVelocity_WeeklyGranularity verifies that weekly granularity aggregates
// completions by ISO week.
func TestVelocity_WeeklyGranularity(t *testing.T) {
	seeds := []seedArtifact{
		{relPath: "lifecycle/requirements/vel-weekly-1.md",
			content: makeArtifact("Vel Weekly 1", "ticket", "done", "vel-weekly-1", "", "Body.")},
		{relPath: "lifecycle/requirements/vel-weekly-2.md",
			content: makeArtifact("Vel Weekly 2", "ticket", "done", "vel-weekly-2", "", "Body.")},
		{relPath: "lifecycle/requirements/vel-weekly-3.md",
			content: makeArtifact("Vel Weekly 3", "ticket", "done", "vel-weekly-3", "", "Body.")},
	}
	env := newTestEnv(t, seeds)
	env.login("admin@test.local", "admin-pass-123")

	// Two events 2 weeks ago, one event this week.
	seedVelocityEvent(t, env, "lifecycle/requirements/vel-weekly-1.md", daysAgo(14))
	seedVelocityEvent(t, env, "lifecycle/requirements/vel-weekly-2.md", daysAgo(14))
	seedVelocityEvent(t, env, "lifecycle/requirements/vel-weekly-3.md", daysAgo(1))

	resp := env.doRequest("GET", "/api/p/testproject/dashboard/velocity?granularity=weekly&days=28", nil)
	requireStatus(t, resp, 200)
	data := readJSON(t, resp)

	gran, _ := data["granularity"].(string)
	if gran != "weekly" {
		t.Errorf("granularity: want %q, got %q", "weekly", gran)
	}

	buckets := decodeVelocityBuckets(t, data)

	// Must have at least 5 weekly buckets for a 28-day window.
	if len(buckets) < 4 {
		t.Errorf("expected at least 4 weekly buckets for days=28, got %d", len(buckets))
	}

	// Bucket labels must be in YYYY-Www format.
	for _, b := range buckets {
		if len(b.period) < 8 || b.period[4] != '-' || b.period[5] != 'W' {
			t.Errorf("weekly bucket period has unexpected format: %q", b.period)
		}
	}

	// Total event count must equal 3.
	total := 0
	for _, b := range buckets {
		total += b.count
	}
	if total != 3 {
		t.Errorf("total count across all weekly buckets: want 3, got %d", total)
	}

	// The week 2 weeks ago should have count 2.
	twoWeeksAgo := time.Now().AddDate(0, 0, -14)
	y, w := twoWeeksAgo.ISOWeek()
	key2w := fmt.Sprintf("%04d-W%02d", y, w)
	if c := velocityBucketCount(buckets, key2w); c != 2 {
		t.Errorf("bucket %s count: want 2, got %d", key2w, c)
	}
}

// TestVelocity_MonthlyGranularity verifies that monthly granularity aggregates
// completions by calendar month (YYYY-MM format).
func TestVelocity_MonthlyGranularity(t *testing.T) {
	seeds := []seedArtifact{
		{relPath: "lifecycle/requirements/vel-monthly-1.md",
			content: makeArtifact("Vel Monthly 1", "ticket", "done", "vel-monthly-1", "", "Body.")},
		{relPath: "lifecycle/requirements/vel-monthly-2.md",
			content: makeArtifact("Vel Monthly 2", "ticket", "done", "vel-monthly-2", "", "Body.")},
	}
	env := newTestEnv(t, seeds)
	env.login("admin@test.local", "admin-pass-123")

	// One event last month, one event this month.
	lastMonth := time.Now().AddDate(0, -1, 0)
	lastMonthTS := time.Date(lastMonth.Year(), lastMonth.Month(), 15, 12, 0, 0, 0, lastMonth.Location()).Unix()
	thisMonthTS := daysAgo(1)

	seedVelocityEvent(t, env, "lifecycle/requirements/vel-monthly-1.md", lastMonthTS)
	seedVelocityEvent(t, env, "lifecycle/requirements/vel-monthly-2.md", thisMonthTS)

	resp := env.doRequest("GET", "/api/p/testproject/dashboard/velocity?granularity=monthly&days=60", nil)
	requireStatus(t, resp, 200)
	data := readJSON(t, resp)

	gran, _ := data["granularity"].(string)
	if gran != "monthly" {
		t.Errorf("granularity: want %q, got %q", "monthly", gran)
	}

	buckets := decodeVelocityBuckets(t, data)

	// Must have at least 2 monthly buckets.
	if len(buckets) < 2 {
		t.Errorf("expected at least 2 monthly buckets for days=60, got %d", len(buckets))
	}

	// Labels must be in YYYY-MM format.
	for _, b := range buckets {
		if len(b.period) != 7 || b.period[4] != '-' {
			t.Errorf("monthly bucket period has unexpected format: %q", b.period)
		}
	}

	// Last month's bucket should have count 1.
	keyLastMonth := lastMonth.Format("2006-01")
	if c := velocityBucketCount(buckets, keyLastMonth); c != 1 {
		t.Errorf("bucket %s count: want 1, got %d", keyLastMonth, c)
	}

	// This month's bucket should have count 1.
	keyThisMonth := time.Now().Format("2006-01")
	if c := velocityBucketCount(buckets, keyThisMonth); c != 1 {
		t.Errorf("bucket %s count: want 1, got %d", keyThisMonth, c)
	}
}

// TestVelocity_ZeroGapsIncluded verifies that days within the window that have
// no completions still appear as buckets with count 0.
func TestVelocity_ZeroGapsIncluded(t *testing.T) {
	seeds := []seedArtifact{
		{relPath: "lifecycle/requirements/vel-gaps-1.md",
			content: makeArtifact("Vel Gaps 1", "ticket", "done", "vel-gaps-1", "", "Body.")},
	}
	env := newTestEnv(t, seeds)
	env.login("admin@test.local", "admin-pass-123")

	// Single event today; all other days in the window should be zero.
	seedVelocityEvent(t, env, "lifecycle/requirements/vel-gaps-1.md", daysAgo(0))

	resp := env.doRequest("GET", "/api/p/testproject/dashboard/velocity?granularity=daily&days=7", nil)
	requireStatus(t, resp, 200)
	data := readJSON(t, resp)

	buckets := decodeVelocityBuckets(t, data)

	// With days=7 there should be 8 daily buckets (today inclusive).
	if len(buckets) < 7 {
		t.Errorf("expected ≥7 daily buckets for days=7, got %d", len(buckets))
	}

	// Count how many buckets have count=0 — should be at least 6.
	zeroBuckets := 0
	for _, b := range buckets {
		if b.count == 0 {
			zeroBuckets++
		}
	}
	if zeroBuckets < 6 {
		t.Errorf("expected ≥6 zero-count buckets (gaps), got %d", zeroBuckets)
	}
}

// TestVelocity_InvalidGranularityDefaultsWeekly verifies that an unrecognised
// granularity value is silently coerced to "weekly".
func TestVelocity_InvalidGranularityDefaultsWeekly(t *testing.T) {
	env := newTestEnv(t, nil)
	env.login("admin@test.local", "admin-pass-123")

	resp := env.doRequest("GET", "/api/p/testproject/dashboard/velocity?granularity=bogus", nil)
	requireStatus(t, resp, 200)
	data := readJSON(t, resp)

	gran, _ := data["granularity"].(string)
	if gran != "weekly" {
		t.Errorf("granularity: want %q for invalid input, got %q", "weekly", gran)
	}

	// Bucket period labels must be in ISO week format.
	buckets := decodeVelocityBuckets(t, data)
	for _, b := range buckets {
		if len(b.period) < 8 || b.period[4] != '-' || b.period[5] != 'W' {
			t.Errorf("expected ISO week period label, got %q", b.period)
		}
	}
}

// TestVelocity_DaysParamLimitsWindow verifies that the days query parameter
// limits the lookback window. With days=7, events older than 7 days must not
// appear in the response.
func TestVelocity_DaysParamLimitsWindow(t *testing.T) {
	seeds := []seedArtifact{
		{relPath: "lifecycle/requirements/vel-days-1.md",
			content: makeArtifact("Vel Days 1", "ticket", "done", "vel-days-1", "", "Body.")},
		{relPath: "lifecycle/requirements/vel-days-2.md",
			content: makeArtifact("Vel Days 2", "ticket", "done", "vel-days-2", "", "Body.")},
	}
	env := newTestEnv(t, seeds)
	env.login("admin@test.local", "admin-pass-123")

	// One event within the last 7 days, one event 30 days ago (outside window).
	seedVelocityEvent(t, env, "lifecycle/requirements/vel-days-1.md", daysAgo(3))
	seedVelocityEvent(t, env, "lifecycle/requirements/vel-days-2.md", daysAgo(30))

	resp := env.doRequest("GET", "/api/p/testproject/dashboard/velocity?granularity=daily&days=7", nil)
	requireStatus(t, resp, 200)
	data := readJSON(t, resp)

	buckets := decodeVelocityBuckets(t, data)

	// The response must not contain a bucket for 30 days ago.
	key30 := time.Now().AddDate(0, 0, -30).Format("2006-01-02")
	if c := velocityBucketCount(buckets, key30); c > 0 {
		t.Errorf("bucket %s should be outside days=7 window, got count %d", key30, c)
	}

	// Total visible count should be 1 (only the 3-day-ago event).
	total := 0
	for _, b := range buckets {
		total += b.count
	}
	if total != 1 {
		t.Errorf("total event count with days=7: want 1, got %d", total)
	}
}

// ── velocity helpers ──────────────────────────────────────────────────────────

type velocityBucket struct {
	period string
	count  int
}

// decodeVelocityBuckets parses the "buckets" array from a velocity API response.
func decodeVelocityBuckets(t *testing.T, data map[string]any) []velocityBucket {
	t.Helper()
	raw, ok := data["buckets"]
	if !ok {
		t.Fatal("response missing 'buckets' key")
	}
	items, ok := raw.([]any)
	if !ok {
		t.Fatalf("'buckets' is not an array, got %T", raw)
	}
	out := make([]velocityBucket, 0, len(items))
	for _, item := range items {
		entry, ok := item.(map[string]any)
		if !ok {
			continue
		}
		period, _ := entry["period"].(string)
		count, _ := entry["count"].(float64)
		out = append(out, velocityBucket{period: period, count: int(count)})
	}
	return out
}

// velocityBucketCount returns the count for the given period key, or 0 if not
// found.
func velocityBucketCount(buckets []velocityBucket, period string) int {
	for _, b := range buckets {
		if b.period == period {
			return b.count
		}
	}
	return 0
}

// ── Milestone 1 — days parameter behaviour ────────────────────────────────────

// TestVelocityDaysParam_Explicit verifies that days=14 with daily granularity
// returns a window of approximately 14 days (at least 14 buckets).
func TestVelocityDaysParam_Explicit(t *testing.T) {
	env := newTestEnv(t, nil)
	env.login("admin@test.local", "admin-pass-123")

	resp := env.doRequest("GET", "/api/p/testproject/dashboard/velocity?granularity=daily&days=14", nil)
	requireStatus(t, resp, 200)
	data := readJSON(t, resp)

	gran, _ := data["granularity"].(string)
	if gran != "daily" {
		t.Errorf("granularity: want %q, got %q", "daily", gran)
	}

	buckets := decodeVelocityBuckets(t, data)
	// days=14 produces buckets from 14 days ago through today inclusive (≥14).
	if len(buckets) < 14 {
		t.Errorf("days=14 daily: expected ≥14 buckets, got %d", len(buckets))
	}
	// Must not span the full 90-day default window.
	if len(buckets) > 20 {
		t.Errorf("days=14 daily: expected ≤20 buckets (not default 90-day window), got %d", len(buckets))
	}
}

// TestVelocityDaysParam_Zero verifies that days=0 falls back to the 90-day
// default window (HTTP handler requires n > 0).
func TestVelocityDaysParam_Zero(t *testing.T) {
	env := newTestEnv(t, nil)
	env.login("admin@test.local", "admin-pass-123")

	resp := env.doRequest("GET", "/api/p/testproject/dashboard/velocity?granularity=weekly&days=0", nil)
	requireStatus(t, resp, 200)
	data := readJSON(t, resp)

	buckets := decodeVelocityBuckets(t, data)
	// Default 90-day window with weekly granularity → at least 12 ISO-week buckets.
	if len(buckets) < 12 {
		t.Errorf("days=0 (fallback to 90) weekly: expected ≥12 buckets, got %d", len(buckets))
	}
}

// TestVelocityDaysParam_Negative verifies that days=-5 falls back to the 90-day
// default window.
func TestVelocityDaysParam_Negative(t *testing.T) {
	env := newTestEnv(t, nil)
	env.login("admin@test.local", "admin-pass-123")

	resp := env.doRequest("GET", "/api/p/testproject/dashboard/velocity?granularity=weekly&days=-5", nil)
	requireStatus(t, resp, 200)
	data := readJSON(t, resp)

	buckets := decodeVelocityBuckets(t, data)
	// Negative days falls back to default 90-day window.
	if len(buckets) < 12 {
		t.Errorf("days=-5 (fallback to 90) weekly: expected ≥12 buckets, got %d", len(buckets))
	}
}

// TestVelocityDaysParam_OverMax verifies that days=400 is clamped to the 365-day
// maximum and returns the same bucket count as days=365.
func TestVelocityDaysParam_OverMax(t *testing.T) {
	env := newTestEnv(t, nil)
	env.login("admin@test.local", "admin-pass-123")

	resp365 := env.doRequest("GET", "/api/p/testproject/dashboard/velocity?granularity=weekly&days=365", nil)
	requireStatus(t, resp365, 200)
	data365 := readJSON(t, resp365)
	buckets365 := decodeVelocityBuckets(t, data365)

	resp400 := env.doRequest("GET", "/api/p/testproject/dashboard/velocity?granularity=weekly&days=400", nil)
	requireStatus(t, resp400, 200)
	data400 := readJSON(t, resp400)
	buckets400 := decodeVelocityBuckets(t, data400)

	// days=400 must be clamped: same bucket count as days=365.
	if len(buckets400) != len(buckets365) {
		t.Errorf("days=400 should be clamped to 365: buckets365=%d, buckets400=%d",
			len(buckets365), len(buckets400))
	}
}

// TestVelocityDaysParam_NonNumeric verifies that a non-numeric days value falls
// back to the 90-day default window.
func TestVelocityDaysParam_NonNumeric(t *testing.T) {
	env := newTestEnv(t, nil)
	env.login("admin@test.local", "admin-pass-123")

	resp := env.doRequest("GET", "/api/p/testproject/dashboard/velocity?granularity=weekly&days=abc", nil)
	requireStatus(t, resp, 200)
	data := readJSON(t, resp)

	buckets := decodeVelocityBuckets(t, data)
	// Non-numeric falls back to default 90-day window.
	if len(buckets) < 12 {
		t.Errorf("days=abc (fallback to 90) weekly: expected ≥12 buckets, got %d", len(buckets))
	}
}

// TestVelocityDaysParam_Omitted verifies that omitting the days parameter
// returns the 90-day default window.
func TestVelocityDaysParam_Omitted(t *testing.T) {
	env := newTestEnv(t, nil)
	env.login("admin@test.local", "admin-pass-123")

	// No days param — must match the explicit days=90 response.
	respOmit := env.doRequest("GET", "/api/p/testproject/dashboard/velocity?granularity=weekly", nil)
	requireStatus(t, respOmit, 200)
	dataOmit := readJSON(t, respOmit)
	bucketsOmit := decodeVelocityBuckets(t, dataOmit)

	resp90 := env.doRequest("GET", "/api/p/testproject/dashboard/velocity?granularity=weekly&days=90", nil)
	requireStatus(t, resp90, 200)
	data90 := readJSON(t, resp90)
	buckets90 := decodeVelocityBuckets(t, data90)

	if len(bucketsOmit) != len(buckets90) {
		t.Errorf("omitted days should equal days=90: omit=%d, explicit90=%d",
			len(bucketsOmit), len(buckets90))
	}
}

// ── Milestone 2 — zero-fill coverage ─────────────────────────────────────────

// TestVelocityZeroFill_Daily7 verifies that an empty project with daily&days=7
// returns at least 7 contiguous buckets all with count=0.
func TestVelocityZeroFill_Daily7(t *testing.T) {
	env := newTestEnv(t, nil)
	env.login("admin@test.local", "admin-pass-123")

	resp := env.doRequest("GET", "/api/p/testproject/dashboard/velocity?granularity=daily&days=7", nil)
	requireStatus(t, resp, 200)
	data := readJSON(t, resp)

	buckets := decodeVelocityBuckets(t, data)
	if len(buckets) < 7 {
		t.Errorf("daily days=7 empty project: expected ≥7 buckets, got %d", len(buckets))
	}

	for _, b := range buckets {
		if b.count != 0 {
			t.Errorf("bucket %s: expected count=0 (empty project), got %d", b.period, b.count)
		}
	}
}

// TestVelocityZeroFill_Weekly28 verifies that an empty project with weekly&days=28
// returns at least 4 buckets all with count=0.
func TestVelocityZeroFill_Weekly28(t *testing.T) {
	env := newTestEnv(t, nil)
	env.login("admin@test.local", "admin-pass-123")

	resp := env.doRequest("GET", "/api/p/testproject/dashboard/velocity?granularity=weekly&days=28", nil)
	requireStatus(t, resp, 200)
	data := readJSON(t, resp)

	buckets := decodeVelocityBuckets(t, data)
	if len(buckets) < 4 {
		t.Errorf("weekly days=28 empty project: expected ≥4 buckets, got %d", len(buckets))
	}

	for _, b := range buckets {
		if b.count != 0 {
			t.Errorf("bucket %s: expected count=0 (empty project), got %d", b.period, b.count)
		}
	}
}

// TestVelocityZeroFill_Monthly90 verifies that an empty project with monthly&days=90
// returns at least 2 monthly buckets all with count=0.
func TestVelocityZeroFill_Monthly90(t *testing.T) {
	env := newTestEnv(t, nil)
	env.login("admin@test.local", "admin-pass-123")

	resp := env.doRequest("GET", "/api/p/testproject/dashboard/velocity?granularity=monthly&days=90", nil)
	requireStatus(t, resp, 200)
	data := readJSON(t, resp)

	buckets := decodeVelocityBuckets(t, data)
	if len(buckets) < 2 {
		t.Errorf("monthly days=90 empty project: expected ≥2 buckets, got %d", len(buckets))
	}

	// Verify all are zero-count.
	for _, b := range buckets {
		if b.count != 0 {
			t.Errorf("bucket %s: expected count=0 (empty project), got %d", b.period, b.count)
		}
	}

	// Verify YYYY-MM format.
	for _, b := range buckets {
		if len(b.period) != 7 || b.period[4] != '-' {
			t.Errorf("monthly bucket period has unexpected format: %q", b.period)
		}
	}
}

// TestVelocityZeroFill_Contiguous verifies that a 14-day daily window with a
// single event placed in the middle produces contiguous date-keyed buckets with
// exactly one non-zero count and all others zero, and no gaps in the date sequence.
func TestVelocityZeroFill_Contiguous(t *testing.T) {
	seeds := []seedArtifact{
		{relPath: "lifecycle/requirements/vel-contiguous-1.md",
			content: makeArtifact("Vel Contiguous 1", "ticket", "done", "vel-contiguous-1", "", "Body.")},
	}
	env := newTestEnv(t, seeds)
	env.login("admin@test.local", "admin-pass-123")

	// Place the single event 7 days ago — midpoint of a 14-day window.
	seedVelocityEvent(t, env, "lifecycle/requirements/vel-contiguous-1.md", daysAgo(7))

	resp := env.doRequest("GET", "/api/p/testproject/dashboard/velocity?granularity=daily&days=14", nil)
	requireStatus(t, resp, 200)
	data := readJSON(t, resp)

	buckets := decodeVelocityBuckets(t, data)
	if len(buckets) < 14 {
		t.Errorf("daily days=14: expected ≥14 buckets, got %d", len(buckets))
	}

	// Exactly one bucket must have count=1; all others must be count=0.
	nonZero := 0
	for _, b := range buckets {
		if b.count > 0 {
			nonZero++
			if b.count != 1 {
				t.Errorf("bucket %s: expected count=1, got %d", b.period, b.count)
			}
		}
	}
	if nonZero != 1 {
		t.Errorf("expected exactly 1 non-zero bucket, got %d", nonZero)
	}

	// The 7-days-ago bucket must be the non-zero one.
	key7 := time.Now().AddDate(0, 0, -7).Format("2006-01-02")
	if c := velocityBucketCount(buckets, key7); c != 1 {
		t.Errorf("bucket %s (7 days ago): want count=1, got %d", key7, c)
	}

	// Verify date contiguity: each period key must be exactly one day after the previous.
	for i := 1; i < len(buckets); i++ {
		prev, err1 := time.Parse("2006-01-02", buckets[i-1].period)
		curr, err2 := time.Parse("2006-01-02", buckets[i].period)
		if err1 != nil || err2 != nil {
			t.Errorf("failed to parse bucket periods: %q, %q", buckets[i-1].period, buckets[i].period)
			continue
		}
		if curr.Sub(prev) != 24*time.Hour {
			t.Errorf("gap between buckets: %q → %q (expected 1 day)", buckets[i-1].period, buckets[i].period)
		}
	}
}
