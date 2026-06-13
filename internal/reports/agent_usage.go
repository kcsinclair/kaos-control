// SPDX-License-Identifier: AGPL-3.0-or-later

// Package reports implements aggregation functions for the analytics report
// endpoints. The functions are pure Go and accept a live *index.Index so the
// HTTP handler is a thin JSON wrapper around them.
package reports

import (
	"database/sql"
	"fmt"
	"sort"
	"time"

	"github.com/kaos-control/kaos-control/internal/index"
)

// ErrBadFilter is returned when the caller supplies an invalid filter (maps to HTTP 400).
type ErrBadFilter struct{ Msg string }

func (e ErrBadFilter) Error() string { return e.Msg }

// AgentUsageFilter contains the caller-supplied filter parameters for the report.
type AgentUsageFilter struct {
	From, To time.Time
	Agents   []string // empty = all agents
	Statuses []string // empty = default terminal set
	Bucket   string   // "hour" | "day" | "week"
	Loc      *time.Location
}

var validBuckets = map[string]bool{"hour": true, "day": true, "week": true}

var defaultStatuses = []string{"done", "failed", "killed", "killed-timeout"}

var knownStatuses = map[string]bool{
	"done": true, "failed": true, "killed": true, "killed-timeout": true, "running": true,
}

// AgentUsageReport is the top-level response body.
type AgentUsageReport struct {
	Summary       AgentUsageSummary                    `json:"summary"`
	Series        []AgentUsageSeriesPoint              `json:"series"`
	SeriesByModel map[string][]AgentUsageSeriesPoint   `json:"series_by_model"`
	SeriesByAgent map[string][]AgentUsageSeriesPoint   `json:"series_by_agent,omitempty"`
}

// AgentUsageSummary is the summary block of the report.
type AgentUsageSummary struct {
	Overall  AgentUsageAggregate   `json:"overall"`
	PerModel []AgentUsageByModel   `json:"per_model"`
	PerAgent []AgentUsageByAgent   `json:"per_agent"`
}

// AgentUsageAggregate contains the aggregate fields for the overall, per-model,
// and per-agent dimensions.
type AgentUsageAggregate struct {
	RunCount                  int64   `json:"run_count"`
	SuccessCount              int64   `json:"success_count"`
	FailureCount              int64   `json:"failure_count"`
	MetricsUnavailableCount   int64   `json:"metrics_unavailable_count"`
	TotalCostUSD              float64 `json:"total_cost_usd"`
	TotalInputCostUSD         float64 `json:"total_input_cost_usd"`
	TotalOutputCostUSD        float64 `json:"total_output_cost_usd"`
	TotalDurationMs           int64   `json:"total_duration_ms"`
	TotalInputTokens          int64   `json:"total_input_tokens"`
	TotalCacheCreationTokens  int64   `json:"total_cache_creation_tokens"`
	TotalCacheReadTokens      int64   `json:"total_cache_read_tokens"`
	TotalOutputTokens         int64   `json:"total_output_tokens"`
	MeanDurationMs            float64 `json:"mean_duration_ms"`
	MedianDurationMs          float64 `json:"median_duration_ms"`
	P95DurationMs             float64 `json:"p95_duration_ms"`
	MeanCostUSD               float64 `json:"mean_cost_usd"`
	MeanOutputTokensPerSecond float64 `json:"mean_output_tokens_per_second"`
	MeanTTFTMs                float64 `json:"mean_ttft_ms"`
	P95TTFTMs                 float64 `json:"p95_ttft_ms"`
	CacheHitRatio             float64 `json:"cache_hit_ratio"`
}

// AgentUsageByModel is one entry in the per_model array.
type AgentUsageByModel struct {
	Model string `json:"model"`
	AgentUsageAggregate
}

// AgentUsageByAgent is one entry in the per_agent array.
type AgentUsageByAgent struct {
	AgentName string `json:"agent_name"`
	AgentUsageAggregate
}

// AgentUsageSeriesPoint is one bucket in a time-series array.
type AgentUsageSeriesPoint struct {
	BucketStart               time.Time `json:"bucket_start"`
	RunCount                  int64     `json:"run_count"`
	SuccessCount              int64     `json:"success_count"`
	FailureCount              int64     `json:"failure_count"`
	MeanDurationMs            float64   `json:"mean_duration_ms"`
	MeanCostUSD               float64   `json:"mean_cost_usd"`
	MeanOutputTokensPerSecond float64   `json:"mean_output_tokens_per_second"`
	MeanTTFTMs                float64   `json:"mean_ttft_ms"`
	CacheHitRatio             float64   `json:"cache_hit_ratio"`
}

// accumulator collects raw values for one dimension slice (overall, per-model, per-agent).
type accumulator struct {
	runCount                int64
	successCount            int64
	failureCount            int64
	metricsUnavailableCount int64
	totalCostUSD            float64
	totalInputCostUSD       float64
	totalOutputCostUSD      float64
	totalDurationMs         int64
	totalInputTokens        int64
	totalCacheCreationTokens int64
	totalCacheReadTokens    int64
	totalOutputTokens       int64
	// slices for percentile computation
	durationSlice []int64
	ttftSlice     []int64
	// for mean_output_tokens_per_second: sum over runs with metrics
	outputTokensPerSecSum float64
	outputTokensPerSecN   int64
	// for cache_hit_ratio: aggregated token counts over runs with metrics
	cacheHitNumer int64 // cache_read_tokens
	cacheHitDenom int64 // input_tokens + cache_creation_tokens + cache_read_tokens
	// for mean_ttft
	ttftSum int64
	ttftN   int64
}

func (a *accumulator) add(status string, metricsAvail bool, costUSD float64, durationApiMs int64,
	inputTok, cacheCreate, cacheRead, outputTok int64, ttftMs *int64,
	inputCostUSD, outputCostUSD float64) {

	a.runCount++
	if status == "done" {
		a.successCount++
	} else {
		a.failureCount++
	}

	if !metricsAvail {
		a.metricsUnavailableCount++
		return
	}

	a.totalCostUSD += costUSD
	a.totalInputCostUSD += inputCostUSD
	a.totalOutputCostUSD += outputCostUSD
	a.totalDurationMs += durationApiMs
	a.totalInputTokens += inputTok
	a.totalCacheCreationTokens += cacheCreate
	a.totalCacheReadTokens += cacheRead
	a.totalOutputTokens += outputTok
	a.durationSlice = append(a.durationSlice, durationApiMs)

	if durationApiMs > 0 {
		opsec := float64(outputTok) / (float64(durationApiMs) / 1000.0)
		a.outputTokensPerSecSum += opsec
		a.outputTokensPerSecN++
	}

	denomTokens := inputTok + cacheCreate + cacheRead
	if denomTokens > 0 {
		a.cacheHitNumer += cacheRead
		a.cacheHitDenom += denomTokens
	}

	if ttftMs != nil {
		a.ttftSum += *ttftMs
		a.ttftN++
		a.ttftSlice = append(a.ttftSlice, *ttftMs)
	}
}

func (a *accumulator) toAggregate() AgentUsageAggregate {
	agg := AgentUsageAggregate{
		RunCount:                 a.runCount,
		SuccessCount:             a.successCount,
		FailureCount:             a.failureCount,
		MetricsUnavailableCount:  a.metricsUnavailableCount,
		TotalCostUSD:             a.totalCostUSD,
		TotalInputCostUSD:        a.totalInputCostUSD,
		TotalOutputCostUSD:       a.totalOutputCostUSD,
		TotalDurationMs:          a.totalDurationMs,
		TotalInputTokens:         a.totalInputTokens,
		TotalCacheCreationTokens: a.totalCacheCreationTokens,
		TotalCacheReadTokens:     a.totalCacheReadTokens,
		TotalOutputTokens:        a.totalOutputTokens,
	}

	metricsN := int64(len(a.durationSlice))
	if metricsN > 0 {
		agg.MeanDurationMs = float64(a.totalDurationMs) / float64(metricsN)
		sorted := make([]int64, len(a.durationSlice))
		copy(sorted, a.durationSlice)
		sort.Slice(sorted, func(i, j int) bool { return sorted[i] < sorted[j] })
		agg.MedianDurationMs = float64(percentile(sorted, 50))
		agg.P95DurationMs = float64(percentile(sorted, 95))
		agg.MeanCostUSD = a.totalCostUSD / float64(metricsN)
	}

	if a.outputTokensPerSecN > 0 {
		agg.MeanOutputTokensPerSecond = a.outputTokensPerSecSum / float64(a.outputTokensPerSecN)
	}

	if a.ttftN > 0 {
		agg.MeanTTFTMs = float64(a.ttftSum) / float64(a.ttftN)
		sorted := make([]int64, len(a.ttftSlice))
		copy(sorted, a.ttftSlice)
		sort.Slice(sorted, func(i, j int) bool { return sorted[i] < sorted[j] })
		agg.P95TTFTMs = float64(percentile(sorted, 95))
	}

	if a.cacheHitDenom > 0 {
		agg.CacheHitRatio = float64(a.cacheHitNumer) / float64(a.cacheHitDenom)
	}

	return agg
}

// percentile returns the p-th percentile value from a sorted slice.
func percentile(sorted []int64, p int) int64 {
	if len(sorted) == 0 {
		return 0
	}
	idx := (p * len(sorted)) / 100
	if idx >= len(sorted) {
		idx = len(sorted) - 1
	}
	return sorted[idx]
}

// seriesBucket accumulates series data for a single time bucket.
type seriesBucket struct {
	runCount    int64
	successCount int64
	failureCount int64
	// for mean computations over metrics-available rows
	durationSum       int64
	durationN         int64
	costSum           float64
	costN             int64
	opsecSum          float64
	opsecN            int64
	ttftSum           int64
	ttftN             int64
	cacheHitNumer     int64
	cacheHitDenom     int64
}

func (b *seriesBucket) add(status string, metricsAvail bool, costUSD float64, durationApiMs int64,
	inputTok, cacheCreate, cacheRead, outputTok int64, ttftMs *int64) {

	b.runCount++
	if status == "done" {
		b.successCount++
	} else {
		b.failureCount++
	}
	if !metricsAvail {
		return
	}
	b.durationSum += durationApiMs
	b.durationN++
	b.costSum += costUSD
	b.costN++
	if durationApiMs > 0 {
		b.opsecSum += float64(outputTok) / (float64(durationApiMs) / 1000.0)
		b.opsecN++
	}
	denom := inputTok + cacheCreate + cacheRead
	if denom > 0 {
		b.cacheHitNumer += cacheRead
		b.cacheHitDenom += denom
	}
	if ttftMs != nil {
		b.ttftSum += *ttftMs
		b.ttftN++
	}
}

func (b *seriesBucket) toPoint(start time.Time) AgentUsageSeriesPoint {
	pt := AgentUsageSeriesPoint{
		BucketStart:  start,
		RunCount:     b.runCount,
		SuccessCount: b.successCount,
		FailureCount: b.failureCount,
	}
	if b.durationN > 0 {
		pt.MeanDurationMs = float64(b.durationSum) / float64(b.durationN)
	}
	if b.costN > 0 {
		pt.MeanCostUSD = b.costSum / float64(b.costN)
	}
	if b.opsecN > 0 {
		pt.MeanOutputTokensPerSecond = b.opsecSum / float64(b.opsecN)
	}
	if b.ttftN > 0 {
		pt.MeanTTFTMs = float64(b.ttftSum) / float64(b.ttftN)
	}
	if b.cacheHitDenom > 0 {
		pt.CacheHitRatio = float64(b.cacheHitNumer) / float64(b.cacheHitDenom)
	}
	return pt
}

// zeroPoint returns a series point with all counts zeroed for bucket-gap filling.
func zeroPoint(start time.Time) AgentUsageSeriesPoint {
	return AgentUsageSeriesPoint{BucketStart: start}
}

// BucketStart computes the start of the time bucket containing t using the
// named granularity and timezone. Exported for use in unit tests.
func BucketStart(t time.Time, bucket string, loc *time.Location) time.Time {
	if loc == nil {
		loc = time.UTC
	}
	t = t.In(loc)
	switch bucket {
	case "hour":
		return time.Date(t.Year(), t.Month(), t.Day(), t.Hour(), 0, 0, 0, loc)
	case "week":
		// ISO week: Monday-aligned.
		wd := int(t.Weekday())
		if wd == 0 {
			wd = 7
		}
		day := t.AddDate(0, 0, -(wd - 1))
		return time.Date(day.Year(), day.Month(), day.Day(), 0, 0, 0, 0, loc)
	default: // "day"
		return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, loc)
	}
}

// nextBucket advances a bucket start by one bucket duration.
func nextBucket(t time.Time, bucket string) time.Time {
	switch bucket {
	case "hour":
		return t.Add(time.Hour)
	case "week":
		return t.AddDate(0, 0, 7)
	default: // "day"
		return t.AddDate(0, 0, 1)
	}
}

// BuildAgentUsageReport executes a single SQL query against the index and
// assembles the summary + series report. It returns ErrBadFilter for invalid
// filter parameters (the HTTP handler maps this to 400).
func BuildAgentUsageReport(idx *index.Index, f AgentUsageFilter) (*AgentUsageReport, error) {
	if err := validateFilter(&f); err != nil {
		return nil, err
	}

	loc := f.Loc
	if loc == nil {
		loc = time.UTC
	}

	db := idx.DB()
	query, qargs := buildQuery(f)
	rows, err := db.Query(query, qargs...)
	if err != nil {
		return nil, fmt.Errorf("querying agent_runs: %w", err)
	}
	defer rows.Close()

	overall := &accumulator{}
	modelAccum := map[string]*accumulator{}
	agentAccum := map[string]*accumulator{}
	// bucket → seriesBucket
	buckets := map[time.Time]*seriesBucket{}
	// model → bucket → seriesBucket
	modelBuckets := map[string]map[time.Time]*seriesBucket{}
	// agent → bucket → seriesBucket
	agentBuckets := map[string]map[time.Time]*seriesBucket{}

	for rows.Next() {
		var (
			runID         string
			agentName     string
			modelN        sql.NullString
			startedAtUnix int64
			status        string
			ttftMsN       sql.NullInt64
			costN         sql.NullFloat64
			durApiMsN     sql.NullInt64
			inputTokN     sql.NullInt64
			cacheCreateN  sql.NullInt64
			cacheReadN    sql.NullInt64
			outputTokN    sql.NullInt64
			metricsAvail  int
		)
		if err := rows.Scan(
			&runID, &agentName, &modelN, &startedAtUnix, &status,
			&ttftMsN, &costN, &durApiMsN,
			&inputTokN, &cacheCreateN, &cacheReadN, &outputTokN,
			&metricsAvail,
		); err != nil {
			return nil, fmt.Errorf("scanning row: %w", err)
		}

		modelStr := ""
		if modelN.Valid {
			modelStr = modelN.String
		}
		if modelStr == "" {
			modelStr = "(unknown)"
		}

		startedAt := time.Unix(startedAtUnix, 0)
		metricsOK := metricsAvail == 1

		var costUSD float64
		var durApiMs, inputTok, cacheCreate, cacheRead, outputTok int64
		if metricsOK {
			costUSD = costN.Float64
			durApiMs = durApiMsN.Int64
			inputTok = inputTokN.Int64
			cacheCreate = cacheCreateN.Int64
			cacheRead = cacheReadN.Int64
			outputTok = outputTokN.Int64
		}
		var ttftPtr *int64
		if ttftMsN.Valid {
			v := ttftMsN.Int64
			ttftPtr = &v
		}

		// Split the recorded total into input-side and output cost using the
		// model's list prices (reconciled to the recorded total). Zero for
		// unknown models or rows without metrics.
		inCost, outCost := splitCost(modelStr, inputTok, cacheCreate, cacheRead, outputTok, costUSD)

		// Overall accumulator.
		overall.add(status, metricsOK, costUSD, durApiMs, inputTok, cacheCreate, cacheRead, outputTok, ttftPtr, inCost, outCost)

		// Per-model accumulator.
		if _, ok := modelAccum[modelStr]; !ok {
			modelAccum[modelStr] = &accumulator{}
		}
		modelAccum[modelStr].add(status, metricsOK, costUSD, durApiMs, inputTok, cacheCreate, cacheRead, outputTok, ttftPtr, inCost, outCost)

		// Per-agent accumulator.
		if _, ok := agentAccum[agentName]; !ok {
			agentAccum[agentName] = &accumulator{}
		}
		agentAccum[agentName].add(status, metricsOK, costUSD, durApiMs, inputTok, cacheCreate, cacheRead, outputTok, ttftPtr, inCost, outCost)

		// Bucket key.
		bk := BucketStart(startedAt, f.Bucket, loc)

		if _, ok := buckets[bk]; !ok {
			buckets[bk] = &seriesBucket{}
		}
		buckets[bk].add(status, metricsOK, costUSD, durApiMs, inputTok, cacheCreate, cacheRead, outputTok, ttftPtr)

		if _, ok := modelBuckets[modelStr]; !ok {
			modelBuckets[modelStr] = map[time.Time]*seriesBucket{}
		}
		if _, ok := modelBuckets[modelStr][bk]; !ok {
			modelBuckets[modelStr][bk] = &seriesBucket{}
		}
		modelBuckets[modelStr][bk].add(status, metricsOK, costUSD, durApiMs, inputTok, cacheCreate, cacheRead, outputTok, ttftPtr)

		if len(f.Agents) > 0 {
			if _, ok := agentBuckets[agentName]; !ok {
				agentBuckets[agentName] = map[time.Time]*seriesBucket{}
			}
			if _, ok := agentBuckets[agentName][bk]; !ok {
				agentBuckets[agentName][bk] = &seriesBucket{}
			}
			agentBuckets[agentName][bk].add(status, metricsOK, costUSD, durApiMs, inputTok, cacheCreate, cacheRead, outputTok, ttftPtr)
		}
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating rows: %w", err)
	}

	// Build the continuous bucket sequence spanning [from, to].
	allBuckets := bucketSequence(f.From, f.To, f.Bucket, loc)

	// Overall series.
	series := make([]AgentUsageSeriesPoint, len(allBuckets))
	for i, bk := range allBuckets {
		if b, ok := buckets[bk]; ok {
			series[i] = b.toPoint(bk)
		} else {
			series[i] = zeroPoint(bk)
		}
	}

	// Per-model series.
	seriesByModel := map[string][]AgentUsageSeriesPoint{}
	for modelStr, mb := range modelBuckets {
		pts := make([]AgentUsageSeriesPoint, len(allBuckets))
		for i, bk := range allBuckets {
			if b, ok := mb[bk]; ok {
				pts[i] = b.toPoint(bk)
			} else {
				pts[i] = zeroPoint(bk)
			}
		}
		seriesByModel[modelStr] = pts
	}

	// Per-agent series (only when agent filter is active).
	var seriesByAgent map[string][]AgentUsageSeriesPoint
	if len(f.Agents) > 0 {
		seriesByAgent = map[string][]AgentUsageSeriesPoint{}
		for agentName, ab := range agentBuckets {
			pts := make([]AgentUsageSeriesPoint, len(allBuckets))
			for i, bk := range allBuckets {
				if b, ok := ab[bk]; ok {
					pts[i] = b.toPoint(bk)
				} else {
					pts[i] = zeroPoint(bk)
				}
			}
			seriesByAgent[agentName] = pts
		}
	}

	// Build per-model summary, sorted by model name for stable output.
	modelNames := make([]string, 0, len(modelAccum))
	for m := range modelAccum {
		modelNames = append(modelNames, m)
	}
	sort.Strings(modelNames)
	perModel := make([]AgentUsageByModel, 0, len(modelNames))
	for _, m := range modelNames {
		perModel = append(perModel, AgentUsageByModel{
			Model:               m,
			AgentUsageAggregate: modelAccum[m].toAggregate(),
		})
	}

	// Build per-agent summary, sorted by agent name.
	agentNames := make([]string, 0, len(agentAccum))
	for a := range agentAccum {
		agentNames = append(agentNames, a)
	}
	sort.Strings(agentNames)
	perAgent := make([]AgentUsageByAgent, 0, len(agentNames))
	for _, a := range agentNames {
		perAgent = append(perAgent, AgentUsageByAgent{
			AgentName:           a,
			AgentUsageAggregate: agentAccum[a].toAggregate(),
		})
	}

	return &AgentUsageReport{
		Summary: AgentUsageSummary{
			Overall:  overall.toAggregate(),
			PerModel: perModel,
			PerAgent: perAgent,
		},
		Series:        series,
		SeriesByModel: seriesByModel,
		SeriesByAgent: seriesByAgent,
	}, nil
}

// validateFilter checks filter fields and fills defaults.
func validateFilter(f *AgentUsageFilter) error {
	if f.Loc == nil {
		f.Loc = time.UTC
	}
	if f.Bucket == "" {
		f.Bucket = "day"
	}
	if !validBuckets[f.Bucket] {
		return ErrBadFilter{Msg: fmt.Sprintf("invalid bucket %q; must be hour, day, or week", f.Bucket)}
	}
	if !f.To.IsZero() && f.From.After(f.To) {
		return ErrBadFilter{Msg: "to before from"}
	}
	for _, s := range f.Statuses {
		if !knownStatuses[s] {
			return ErrBadFilter{Msg: fmt.Sprintf("unknown status %q", s)}
		}
	}
	return nil
}

// buildQuery constructs the parameterised SELECT for the given filter.
func buildQuery(f AgentUsageFilter) (string, []any) {
	statuses := f.Statuses
	if len(statuses) == 0 {
		statuses = defaultStatuses
	}

	args := []any{f.From.Unix(), f.To.Unix()}

	// Build status IN clause.
	statusIn := "("
	for i, s := range statuses {
		if i > 0 {
			statusIn += ","
		}
		statusIn += "?"
		args = append(args, s)
	}
	statusIn += ")"

	agentFilter := ""
	if len(f.Agents) > 0 {
		agentFilter = " AND agent_name IN ("
		for i, a := range f.Agents {
			if i > 0 {
				agentFilter += ","
			}
			agentFilter += "?"
			args = append(args, a)
		}
		agentFilter += ")"
	}

	q := `SELECT run_id, agent_name, model, started_at, status,
		         ttft_ms, total_cost_usd, duration_api_ms,
		         input_tokens, cache_creation_tokens, cache_read_tokens, output_tokens,
		         COALESCE(metrics_available, 0)
		  FROM agent_runs
		  WHERE started_at BETWEEN ? AND ?
		    AND status IN ` + statusIn + agentFilter

	return q, args
}

// bucketSequence returns all bucket-start times spanning [from, to] inclusive.
func bucketSequence(from, to time.Time, bucket string, loc *time.Location) []time.Time {
	if loc == nil {
		loc = time.UTC
	}
	start := BucketStart(from, bucket, loc)
	var seq []time.Time
	for !start.After(to) {
		seq = append(seq, start)
		start = nextBucket(start, bucket)
	}
	return seq
}
