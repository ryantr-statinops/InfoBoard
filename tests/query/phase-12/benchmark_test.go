package phase12

import (
	"context"
	"encoding/json"
	"sort"
	"testing"
	"time"

	"github.com/ryantr-statinops/InfoBoard/host/internal/protocol"
)

type renderRow struct {
	ResultID     string
	TabID        int64
	WindowID     int64
	GroupID      *int64
	Title        string
	Domain       string
	WindowLabel  *string
	GroupLabel   *string
	Pinned       bool
	Score        int64
	BestClass    string
	Explanations []string
}

func renderRows(rows []protocol.ResultRow) []renderRow {
	rendered := make([]renderRow, 0, len(rows))
	for _, row := range rows {
		explanations := make([]string, len(row.Explanation))
		copy(explanations, row.Explanation)
		rendered = append(rendered, renderRow{
			ResultID:     row.ResultID,
			TabID:        row.TabID,
			WindowID:     row.WindowID,
			GroupID:      row.GroupID,
			Title:        row.TitleDisplay,
			Domain:       row.DomainDisplay,
			WindowLabel:  row.WindowLabelDisplay,
			GroupLabel:   row.GroupLabelDisplay,
			Pinned:       row.Pinned,
			Score:        row.Score,
			BestClass:    row.BestClass,
			Explanations: explanations,
		})
	}
	return rendered
}

func BenchmarkQuery1000Tabs64CharsWireRender(b *testing.B) {
	fixtures := loadQueryFixtures(b)
	fixture := fixtures["FX-QUERY-1000-TABS-64-CHARS"]
	state := buildFixtureState(b, fixture.Input)
	source := fixtureSource{state: state, ready: state.Revision, readyOK: true}
	service := fixtureService(b, fixture.Input, source)
	request := fixtureRequest(fixture.Input, fixture.Input.Query, 1, fixture.Input.ResultLimit)

	b.ReportAllocs()
	b.ResetTimer()
	durations := make([]time.Duration, 0, b.N)
	wireBytes := 0
	for index := 0; index < b.N; index++ {
		start := time.Now()
		result, err := service.Query(context.Background(), request)
		if err != nil {
			b.Fatalf("query: %v", err)
		}
		payload, failure := protocol.PayloadOf(result)
		if failure != nil {
			b.Fatalf("encode: %v", failure)
		}
		wireBytes = len(payload)
		var decoded protocol.QueryResult
		if err := json.Unmarshal(payload, &decoded); err != nil {
			b.Fatalf("decode: %v", err)
		}
		rendered := renderRows(decoded.Results)
		durations = append(durations, time.Since(start))
		if len(rendered) != 5 {
			b.Fatalf("rendered rows = %d", len(rendered))
		}
	}
	b.StopTimer()
	sort.Slice(durations, func(left, right int) bool {
		return durations[left] < durations[right]
	})
	b.ReportMetric(float64(queryPercentile(durations, 50).Nanoseconds()), "p50-ns")
	b.ReportMetric(float64(queryPercentile(durations, 95).Nanoseconds()), "p95-ns")
	b.ReportMetric(float64(queryPercentile(durations, 99).Nanoseconds()), "p99-ns")
	b.ReportMetric(float64(wireBytes), "wire-bytes")
}

func queryPercentile(values []time.Duration, percent int) time.Duration {
	if len(values) == 0 {
		return 0
	}
	index := (len(values)*percent + 99) / 100
	if index < 1 {
		index = 1
	}
	if index > len(values) {
		index = len(values)
	}
	return values[index-1]
}
