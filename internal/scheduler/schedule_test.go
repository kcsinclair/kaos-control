// SPDX-License-Identifier: AGPL-3.0-or-later

package scheduler

import (
	"testing"
	"time"
)

// ref is a fixed UTC base time used across schedule tests.
var ref = time.Date(2026, 5, 6, 0, 0, 0, 0, time.UTC)

// TestCron5FieldNextFire verifies a 5-field cron fires at the correct next minute
// when now is before the target hour.
func TestCron5FieldNextFire(t *testing.T) {
	// "0 2 * * *" fires at 02:00 daily.
	// now = 2026-05-06T01:00:00Z → next = 2026-05-06T02:00:00Z
	now := ref.Add(time.Hour) // 01:00:00
	spec := ScheduleSpec{Kind: ScheduleKindCron, Cron: "0 2 * * *"}
	next := NextFireTime(spec, time.Time{}, now)
	want := time.Date(2026, 5, 6, 2, 0, 0, 0, time.UTC)
	if !next.Equal(want) {
		t.Errorf("got %v, want %v", next, want)
	}
}

// TestCron5FieldPastToday verifies that when now is past the target hour the next
// fire time is on the following day.
func TestCron5FieldPastToday(t *testing.T) {
	// now = 2026-05-06T03:00:00Z → next = 2026-05-07T02:00:00Z
	now := ref.Add(3 * time.Hour) // 03:00:00
	spec := ScheduleSpec{Kind: ScheduleKindCron, Cron: "0 2 * * *"}
	next := NextFireTime(spec, time.Time{}, now)
	want := time.Date(2026, 5, 7, 2, 0, 0, 0, time.UTC)
	if !next.Equal(want) {
		t.Errorf("got %v, want %v", next, want)
	}
}

// TestCron6FieldWithSeconds verifies a 6-field cron expression fires at the
// correct second within the minute.
func TestCron6FieldWithSeconds(t *testing.T) {
	// "30 0 2 * * *" fires at 02:00:30.
	// now = 2026-05-06T00:00:00Z → next = 2026-05-06T02:00:30Z
	spec := ScheduleSpec{Kind: ScheduleKindCron, Cron: "30 0 2 * * *"}
	next := NextFireTime(spec, time.Time{}, ref)
	want := time.Date(2026, 5, 6, 2, 0, 30, 0, time.UTC)
	if !next.Equal(want) {
		t.Errorf("got %v, want %v", next, want)
	}
}

// TestCronDayOfWeek verifies that day-of-week filtering selects the correct date.
func TestCronDayOfWeek(t *testing.T) {
	// "0 9 * * 1" fires at 09:00 every Monday.
	// 2026-05-06 is a Wednesday. Next Monday is 2026-05-11.
	now := ref // Wednesday
	spec := ScheduleSpec{Kind: ScheduleKindCron, Cron: "0 9 * * 1"}
	next := NextFireTime(spec, time.Time{}, now)
	// 2026-05-11 is a Monday.
	want := time.Date(2026, 5, 11, 9, 0, 0, 0, time.UTC)
	if !next.Equal(want) {
		t.Errorf("got %v, want %v", next, want)
	}
}

// TestIntervalFromLastRun verifies that an interval schedule fires relative to
// the last run end time.
func TestIntervalFromLastRun(t *testing.T) {
	// every: 30m, last run ended at T01:00:00Z, now T01:20:00Z → next T01:30:00Z
	lastRun := ref.Add(time.Hour)            // 01:00:00
	now := ref.Add(time.Hour + 20*time.Minute) // 01:20:00
	spec := ScheduleSpec{Kind: ScheduleKindInterval, Interval: 30 * time.Minute}
	next := NextFireTime(spec, lastRun, now)
	want := ref.Add(time.Hour + 30*time.Minute) // 01:30:00
	if !next.Equal(want) {
		t.Errorf("got %v, want %v", next, want)
	}
}

// TestIntervalNoPriorRun verifies that a first interval-scheduled run fires
// immediately (next = now).
func TestIntervalNoPriorRun(t *testing.T) {
	spec := ScheduleSpec{Kind: ScheduleKindInterval, Interval: 30 * time.Minute}
	next := NextFireTime(spec, time.Time{}, ref)
	if !next.Equal(ref) {
		t.Errorf("got %v, want %v (now, i.e. fire immediately)", next, ref)
	}
}

// TestOnceFuture verifies that a one-off schedule with a future time returns
// that time unchanged.
func TestOnceFuture(t *testing.T) {
	target := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)
	spec := ScheduleSpec{Kind: ScheduleKindOneOff, At: target}
	next := NextFireTime(spec, time.Time{}, ref) // ref is before target
	if !next.Equal(target) {
		t.Errorf("got %v, want %v", next, target)
	}
}

// TestOncePast verifies that a one-off schedule with a past time returns the
// zero value (skip).
func TestOncePast(t *testing.T) {
	past := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	spec := ScheduleSpec{Kind: ScheduleKindOneOff, At: past}
	next := NextFireTime(spec, time.Time{}, ref) // ref is after past
	if !next.IsZero() {
		t.Errorf("expected zero time for past one-off, got %v", next)
	}
}

// TestInvalidCronExpression verifies that parsing a bad cron expression returns
// an error.
func TestInvalidCronExpression(t *testing.T) {
	err := ValidateScheduleSpec(ScheduleSpec{Kind: ScheduleKindCron, Cron: "not a cron"})
	if err == nil {
		t.Error("expected error for invalid cron expression, got nil")
	}
}

// TestInvalidInterval verifies that a non-positive interval returns an error.
func TestInvalidInterval(t *testing.T) {
	err := ValidateScheduleSpec(ScheduleSpec{Kind: ScheduleKindInterval, Interval: 0})
	if err == nil {
		t.Error("expected error for zero interval, got nil")
	}
	err = ValidateScheduleSpec(ScheduleSpec{Kind: ScheduleKindInterval, Interval: -time.Minute})
	if err == nil {
		t.Error("expected error for negative interval, got nil")
	}
}
