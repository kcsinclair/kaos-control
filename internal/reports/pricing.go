// SPDX-License-Identifier: AGPL-3.0-or-later

package reports

import "strings"

// ModelPrice holds per-million-token USD list prices for a model family.
// CacheWrite is the 5-minute ("ephemeral") cache-write price; CacheRead is the
// cache-hit read price.
//
// These are list prices, verified against recorded run costs in
// pricing_test.go (token-count × price reproduces the API-reported costUSD to
// the cent). They MUST be updated when Anthropic changes pricing or ships a new
// model family — an unknown model yields no cost split rather than wrong
// numbers (see splitCost).
type ModelPrice struct {
	InputPerMTok      float64
	OutputPerMTok     float64
	CacheWritePerMTok float64
	CacheReadPerMTok  float64
}

// modelPrices maps a model-family name prefix to its list prices. Lookups match
// by prefix so dated / versioned ids (claude-sonnet-4-6,
// claude-haiku-4-5-20251001, claude-opus-4-7, …) all resolve to their family.
// Order does not matter — the families share no common prefix.
var modelPrices = []struct {
	prefix string
	price  ModelPrice
}{
	{"claude-opus-4", ModelPrice{InputPerMTok: 5, OutputPerMTok: 25, CacheWritePerMTok: 6.25, CacheReadPerMTok: 0.50}},
	{"claude-sonnet-4", ModelPrice{InputPerMTok: 3, OutputPerMTok: 15, CacheWritePerMTok: 3.75, CacheReadPerMTok: 0.30}},
	{"claude-haiku-4", ModelPrice{InputPerMTok: 1, OutputPerMTok: 5, CacheWritePerMTok: 1.25, CacheReadPerMTok: 0.10}},
}

// priceFor returns the list prices for a model and whether a family matched.
func priceFor(model string) (ModelPrice, bool) {
	for _, e := range modelPrices {
		if strings.HasPrefix(model, e.prefix) {
			return e.price, true
		}
	}
	return ModelPrice{}, false
}

// splitCost apportions a run's recorded total cost into input-side and output
// cost from the run's token counts and the model's list prices.
//
//   - "input cost" covers uncached input + cache writes + cache reads
//   - "output cost" covers output tokens
//
// The two components are scaled so they sum exactly to recordedCostUSD — the
// API-reported total is authoritative, so the split always reconciles to it
// even if list prices have drifted since the run. Returns (0, 0) for an unknown
// model or when no priced tokens are present; callers treat that as "no split".
func splitCost(model string, inputTok, cacheWriteTok, cacheReadTok, outputTok int64, recordedCostUSD float64) (inputCostUSD, outputCostUSD float64) {
	p, ok := priceFor(model)
	if !ok {
		return 0, 0
	}
	inCost := (float64(inputTok)*p.InputPerMTok +
		float64(cacheWriteTok)*p.CacheWritePerMTok +
		float64(cacheReadTok)*p.CacheReadPerMTok) / 1e6
	outCost := float64(outputTok) * p.OutputPerMTok / 1e6
	computed := inCost + outCost
	if computed <= 0 {
		return 0, 0
	}
	if recordedCostUSD > 0 {
		scale := recordedCostUSD / computed
		inCost *= scale
		outCost *= scale
	}
	return inCost, outCost
}
