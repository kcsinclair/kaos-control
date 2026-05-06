package scheduler

import (
	"encoding/json"
	"fmt"
	"math/bits"
	"strconv"
	"strings"
	"time"
)

// ParseScheduleSpec parses a JSON-encoded ScheduleSpec and validates it.
func ParseScheduleSpec(raw string) (ScheduleSpec, error) {
	var s ScheduleSpec
	if err := json.Unmarshal([]byte(raw), &s); err != nil {
		return s, fmt.Errorf("invalid schedule JSON: %w", err)
	}
	return s, ValidateScheduleSpec(s)
}

// ValidateScheduleSpec returns an error if the spec is malformed.
func ValidateScheduleSpec(s ScheduleSpec) error {
	switch s.Kind {
	case ScheduleKindCron:
		if s.Cron == "" {
			return fmt.Errorf("cron schedule: cron expression must not be empty")
		}
		if _, err := parseCron(s.Cron); err != nil {
			return fmt.Errorf("cron schedule: %w", err)
		}
	case ScheduleKindInterval:
		if s.Interval <= 0 {
			return fmt.Errorf("interval schedule: interval must be positive")
		}
	case ScheduleKindOneOff:
		if s.At.IsZero() {
			return fmt.Errorf("one_off schedule: at time must be set")
		}
	default:
		return fmt.Errorf("unknown schedule kind %q (want cron, interval, or one_off)", s.Kind)
	}
	return nil
}

// NextFireTime returns the next time a job with spec s should fire, given lastRun
// (zero if the job has never run). Returns zero if the job should not fire again
// (e.g. a past one-off).
func NextFireTime(s ScheduleSpec, lastRun time.Time, now time.Time) time.Time {
	switch s.Kind {
	case ScheduleKindCron:
		c, err := parseCron(s.Cron)
		if err != nil {
			return time.Time{}
		}
		return c.Next(now)
	case ScheduleKindInterval:
		if s.Interval <= 0 {
			return time.Time{}
		}
		if lastRun.IsZero() {
			return now
		}
		return lastRun.Add(s.Interval)
	case ScheduleKindOneOff:
		if s.At.After(now) {
			return s.At
		}
		return time.Time{}
	}
	return time.Time{}
}

// ----- minimal cron parser -----
// Supports 5-field (min hour dom month dow) and 6-field (sec min hour dom month dow).
// Field syntax: *, n, */step, a-b, a-b/step, and comma-separated lists of the above.

type cronSchedule struct {
	second  uint64 // bits 0-59
	minute  uint64 // bits 0-59
	hour    uint64 // bits 0-23
	dom     uint64 // bits 1-31
	month   uint64 // bits 1-12
	dow     uint64 // bits 0-6  (0=Sunday)
	hasSec  bool   // true for 6-field expressions
}

func parseCron(expr string) (*cronSchedule, error) {
	fields := strings.Fields(expr)
	var c cronSchedule
	switch len(fields) {
	case 5:
		// min hour dom month dow
		c.hasSec = false
		c.second = 1 // fire at :00 of the minute
		if err := parseFields(fields, &c, false); err != nil {
			return nil, err
		}
	case 6:
		// sec min hour dom month dow
		c.hasSec = true
		if err := parseFields(fields, &c, true); err != nil {
			return nil, err
		}
	default:
		return nil, fmt.Errorf("cron expression must have 5 or 6 fields, got %d", len(fields))
	}
	return &c, nil
}

func parseFields(fields []string, c *cronSchedule, hasSec bool) error {
	if hasSec {
		sec, err := parseCronField(fields[0], 0, 59)
		if err != nil {
			return fmt.Errorf("second field: %w", err)
		}
		c.second = sec
		fields = fields[1:]
	}
	var err error
	if c.minute, err = parseCronField(fields[0], 0, 59); err != nil {
		return fmt.Errorf("minute field: %w", err)
	}
	if c.hour, err = parseCronField(fields[1], 0, 23); err != nil {
		return fmt.Errorf("hour field: %w", err)
	}
	if c.dom, err = parseCronField(fields[2], 1, 31); err != nil {
		return fmt.Errorf("day-of-month field: %w", err)
	}
	if c.month, err = parseCronField(fields[3], 1, 12); err != nil {
		return fmt.Errorf("month field: %w", err)
	}
	if c.dow, err = parseCronField(fields[4], 0, 6); err != nil {
		return fmt.Errorf("day-of-week field: %w", err)
	}
	return nil
}

// parseCronField parses one cron field (possibly comma-separated) into a bitmask.
func parseCronField(field string, min, max int) (uint64, error) {
	var mask uint64
	for _, part := range strings.Split(field, ",") {
		m, err := parseOnePart(part, min, max)
		if err != nil {
			return 0, err
		}
		mask |= m
	}
	return mask, nil
}

func parseOnePart(part string, min, max int) (uint64, error) {
	step := 1
	if idx := strings.Index(part, "/"); idx >= 0 {
		var err error
		step, err = strconv.Atoi(part[idx+1:])
		if err != nil || step < 1 {
			return 0, fmt.Errorf("invalid step %q", part[idx+1:])
		}
		part = part[:idx]
	}

	var lo, hi int
	if part == "*" {
		lo, hi = min, max
	} else if idx := strings.Index(part, "-"); idx >= 0 {
		var err error
		lo, err = strconv.Atoi(part[:idx])
		if err != nil {
			return 0, fmt.Errorf("invalid range low %q", part[:idx])
		}
		hi, err = strconv.Atoi(part[idx+1:])
		if err != nil {
			return 0, fmt.Errorf("invalid range high %q", part[idx+1:])
		}
	} else {
		v, err := strconv.Atoi(part)
		if err != nil {
			return 0, fmt.Errorf("invalid cron value %q", part)
		}
		lo, hi = v, v
	}
	if lo < min || lo > max || hi < min || hi > max || lo > hi {
		return 0, fmt.Errorf("value %d-%d out of range [%d-%d]", lo, hi, min, max)
	}
	var mask uint64
	for v := lo; v <= hi; v += step {
		mask |= 1 << uint(v)
	}
	return mask, nil
}

// Next returns the earliest time strictly after t that matches c.
// Searches up to 4 years ahead to avoid infinite loops on impossible expressions.
func (c *cronSchedule) Next(t time.Time) time.Time {
	// Advance by one second to avoid re-firing at the same instant.
	t = t.Add(time.Second).Truncate(time.Second)

	deadline := t.Add(4 * 365 * 24 * time.Hour)
	for t.Before(deadline) {
		// Month check (1-based).
		if !bitSet(c.month, t.Month()) {
			// Advance to first day of next month.
			t = time.Date(t.Year(), t.Month()+1, 1, 0, 0, 0, 0, t.Location())
			continue
		}
		// Day-of-month check (1-based).
		if !bitSet(c.dom, t.Day()) {
			t = time.Date(t.Year(), t.Month(), t.Day()+1, 0, 0, 0, 0, t.Location())
			continue
		}
		// Day-of-week check (0=Sunday).
		if !bitSet(c.dow, t.Weekday()) {
			t = time.Date(t.Year(), t.Month(), t.Day()+1, 0, 0, 0, 0, t.Location())
			continue
		}
		// Hour check.
		if !bitSet64(c.hour, t.Hour()) {
			// Find next matching hour.
			nextH := nextBit64(c.hour, t.Hour()+1, 23)
			if nextH < 0 {
				t = time.Date(t.Year(), t.Month(), t.Day()+1, 0, 0, 0, 0, t.Location())
				continue
			}
			t = time.Date(t.Year(), t.Month(), t.Day(), nextH, 0, 0, 0, t.Location())
			continue
		}
		// Minute check.
		if !bitSet64(c.minute, t.Minute()) {
			nextM := nextBit64(c.minute, t.Minute()+1, 59)
			if nextM < 0 {
				// Roll to next hour.
				t = time.Date(t.Year(), t.Month(), t.Day(), t.Hour()+1, 0, 0, 0, t.Location())
				continue
			}
			t = time.Date(t.Year(), t.Month(), t.Day(), t.Hour(), nextM, 0, 0, t.Location())
			continue
		}
		// Second check.
		if !bitSet64(c.second, t.Second()) {
			nextS := nextBit64(c.second, t.Second()+1, 59)
			if nextS < 0 {
				// Roll to next minute.
				t = time.Date(t.Year(), t.Month(), t.Day(), t.Hour(), t.Minute()+1, 0, 0, t.Location())
				continue
			}
			t = time.Date(t.Year(), t.Month(), t.Day(), t.Hour(), t.Minute(), nextS, 0, t.Location())
			continue
		}
		return t
	}
	return time.Time{}
}

// bitSet returns true if bit v is set in mask (for time.Month / time.Weekday).
func bitSet[T ~int](mask uint64, v T) bool {
	u := int(v)
	if u < 0 || u >= bits.UintSize {
		return false
	}
	return mask>>uint(u)&1 == 1
}

// bitSet64 returns true if bit v is set in mask.
func bitSet64(mask uint64, v int) bool {
	if v < 0 || v >= 64 {
		return false
	}
	return mask>>uint(v)&1 == 1
}

// nextBit64 returns the lowest set bit in mask that is >= from, up to max.
// Returns -1 if none found.
func nextBit64(mask uint64, from, max int) int {
	for v := from; v <= max; v++ {
		if bitSet64(mask, v) {
			return v
		}
	}
	return -1
}
