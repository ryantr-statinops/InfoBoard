package phase10

import (
	"fmt"
	"runtime"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/ryantr-statinops/InfoBoard/host/internal/index"
)

const (
	benchmarkTabs   = 1000
	queryIterations = 300
)

func benchmarkRecords() []index.Record {
	records := make([]index.Record, 0, benchmarkTabs)
	for i := 0; i < benchmarkTabs; i++ {
		records = append(records, index.Record{
			TabID:       int64(i + 1),
			Eligible:    true,
			Pinned:      i%11 == 0,
			Active:      i == 0,
			Title:       fmt.Sprintf("Example Page %d guide documentation", i),
			URL:         fmt.Sprintf("https://example%d.test/guide/start/%d", i%17, i),
			Domain:      fmt.Sprintf("example%d.test", i%17),
			WindowLabel: "Work",
			GroupLabel:  "Research",
		})
	}
	return records
}

func benchmarkQuery() string {
	return strings.TrimSpace(strings.Repeat("example ", 8))
}

func TestLexicalQueryP95Budget(t *testing.T) {
	records := benchmarkRecords()
	state := commit(t, records)
	query := benchmarkQuery()
	if scalars := len([]rune(query)); scalars > 64 {
		t.Fatalf("query has %d scalars, want <= 64", scalars)
	}

	normalizeDurations := make([]time.Duration, 0, queryIterations)
	buildDurations := make([]time.Duration, 0, queryIterations)
	queryDurations := make([]time.Duration, 0, queryIterations)

	for i := 0; i < queryIterations; i++ {
		start := time.Now()
		_, _ = index.PrepareQuery(query)
		normalizeDurations = append(normalizeDurations, time.Since(start))

		start = time.Now()
		_ = commit(t, records)
		buildDurations = append(buildDurations, time.Since(start))

		start = time.Now()
		prepared, err := index.PrepareQuery(query)
		if err != nil {
			t.Fatalf("prepare query: %v", err)
		}
		candidates := candidates(state, prepared.Tokens)
		if len(candidates) != benchmarkTabs {
			t.Fatalf("candidates = %d, want %d", len(candidates), benchmarkTabs)
		}
		queryDurations = append(queryDurations, time.Since(start))
	}

	p50, p95, p99 := percentiles(queryDurations)
	if p95 > 50*time.Millisecond {
		t.Fatalf("query-to-render p95 = %s exceeds 50ms on %s/%s (GOMAXPROCS=%d)", p95, runtime.GOOS, runtime.GOARCH, runtime.GOMAXPROCS(0))
	}
	t.Logf("environment=%s %s/%s GOMAXPROCS=%d", runtime.Version(), runtime.GOOS, runtime.GOARCH, runtime.GOMAXPROCS(0))
	n50, n95, n99 := percentiles(normalizeDurations)
	b50, b95, b99 := percentiles(buildDurations)
	t.Logf("normalize p50=%s p95=%s p99=%s", n50, n95, n99)
	t.Logf("build     p50=%s p95=%s p99=%s", b50, b95, b99)
	t.Logf("query     p50=%s p95=%s p99=%s", p50, p95, p99)
}

func BenchmarkQuery1000Tabs(b *testing.B) {
	state := commitBench(b, benchmarkRecords())
	prepared, err := index.PrepareQuery(benchmarkQuery())
	if err != nil {
		b.Fatalf("prepare query: %v", err)
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if candidates(state, prepared.Tokens) == nil {
			b.Fatal("no candidates")
		}
	}
}

func BenchmarkNormalize1000Tabs(b *testing.B) {
	records := benchmarkRecords()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, record := range records {
			_, _ = index.BuildForm(index.FieldTitle, record.Title, index.MaxTitleScalars)
		}
	}
}

func BenchmarkBuild1000Tabs(b *testing.B) {
	records := benchmarkRecords()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		staged, err := index.Stage("profile", 1, records)
		if err != nil {
			b.Fatalf("stage: %v", err)
		}
		_ = staged.Commit()
	}
}

func candidates(state *index.State, tokens []string) []int64 {
	seen := make(map[int64]struct{}, 64)
	for _, token := range tokens {
		for _, id := range state.Postings[token] {
			seen[id] = struct{}{}
		}
	}
	ids := make([]int64, 0, len(seen))
	for id := range seen {
		ids = append(ids, id)
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	return ids
}

func percentiles(durations []time.Duration) (time.Duration, time.Duration, time.Duration) {
	if len(durations) == 0 {
		return 0, 0, 0
	}
	sorted := append([]time.Duration(nil), durations...)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i] < sorted[j] })
	return sorted[len(sorted)*50/100], sorted[len(sorted)*95/100], sorted[len(sorted)*99/100]
}

func commit(t *testing.T, records []index.Record) *index.State {
	t.Helper()
	staged, err := index.Stage("profile", 1, records)
	if err != nil {
		t.Fatalf("stage: %v", err)
	}
	return staged.Commit()
}

func commitBench(b *testing.B, records []index.Record) *index.State {
	b.Helper()
	staged, err := index.Stage("profile", 1, records)
	if err != nil {
		b.Fatalf("stage: %v", err)
	}
	return staged.Commit()
}
