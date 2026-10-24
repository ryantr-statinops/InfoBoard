package phase11

import (
	"encoding/json"
	"sort"
	"testing"
	"time"

	"github.com/ryantr-statinops/InfoBoard/host/internal/rank"
)

func BenchmarkRank1000Tabs64Chars(b *testing.B) {
	fixtures := loadFixtures(b)
	var input rank.Input
	if err := json.Unmarshal(fixtures["FX-RANKING-1000-TABS"].Input, &input); err != nil {
		b.Fatalf("decode input: %v", err)
	}
	b.ReportAllocs()
	b.ResetTimer()

	durations := make([]time.Duration, 0, b.N)
	for index := 0; index < b.N; index++ {
		start := time.Now()
		output, err := rank.Rank(input)
		durations = append(durations, time.Since(start))
		if err != nil {
			b.Fatalf("rank: %v", err)
		}
		if len(output.Results) != rank.MaxResultLimit {
			b.Fatalf("results = %d", len(output.Results))
		}
	}
	b.StopTimer()
	sort.Slice(durations, func(left, right int) bool {
		return durations[left] < durations[right]
	})
	b.ReportMetric(float64(percentile(durations, 50).Nanoseconds()), "p50-ns")
	b.ReportMetric(float64(percentile(durations, 95).Nanoseconds()), "p95-ns")
	b.ReportMetric(float64(percentile(durations, 99).Nanoseconds()), "p99-ns")
}

func percentile(values []time.Duration, percent int) time.Duration {
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
