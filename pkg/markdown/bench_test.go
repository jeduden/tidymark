package markdown

import (
	"fmt"
	"sort"
	"strings"
	"testing"
	"time"
)

// pkg/markdown is mdsmith's public parse/produce surface and a
// cross-system compatibility contract, so it gets the same kind of
// absolute-budget p95 gate as `mdsmith check` (internal/engine,
// plan 175) and the LSP (internal/lsp, plan 121). The gate guards the
// canonical parser — crucially the mdsmith-specific <?...?>
// processing-instruction block parser, which is the reason this
// package exists over vanilla goldmark.
//
// Two tiers catch two regressions, mirroring the check gate:
//
//   - Small (~150-line doc) — per-parse fixed overhead: front-matter
//     split, parser-pool Get/Put, AST setup. A regression in any of
//     those shows here even though the absolute time is tiny.
//   - Large (~3000-line doc) — parse scaling. A superlinear regression
//     in a block parser or the PI scan (an accidental O(n^2)) barely
//     moves Small but blows Large past its budget.
//
// Budgets are generous (~15-20x the local baseline) so shared CI
// jitter does not flake them; they trip on order-of-magnitude
// regressions, not micro-noise. p95 and per-KB cost are reported as
// metrics so trends stay visible in the job log.
//
// Local baseline (Intel Xeon CI-class box, 2026-05, post pkg/markdown
// extraction #343): Small p95 ~0.23 ms, Large p95 ~4 ms. Budgets are
// set well above that (Small 10 ms ≈40x, Large 100 ms ≈25x) for the
// reasons above.
const (
	parseDocLinesSmall = 150
	parseDocLinesLarge = 3000
)

func BenchmarkParseSmall(b *testing.B) {
	benchParse(b, parseDocLinesSmall, 10*time.Millisecond)
}

func BenchmarkParseLarge(b *testing.B) {
	benchParse(b, parseDocLinesLarge, 100*time.Millisecond)
}

func benchParse(b *testing.B, lines int, budget time.Duration) {
	b.Helper()
	if testing.Short() {
		b.Skip("benchmark skipped in -short mode")
	}
	src := []byte(buildParseDoc(lines))
	b.SetBytes(int64(len(src)))

	// Warm one parse so the parser pool is populated and first-touch
	// allocations are not charged to the first sample.
	_ = Parse(src)

	samples := make([]time.Duration, 0, b.N)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		start := time.Now()
		_ = Parse(src)
		samples = append(samples, time.Since(start))
	}
	b.StopTimer()

	if len(samples) == 0 {
		b.Skip("no samples — benchmark needs more iterations")
	}
	p95 := percentileDur(samples, 0.95)
	b.ReportMetric(float64(p95.Microseconds())/1000, "p95_ms")
	b.ReportMetric(float64(p95.Microseconds())/(float64(len(src))/1024), "us_per_KB")
	if p95 > budget {
		b.Fatalf("parse p95 %v exceeds budget %v for %d-line doc (%d bytes)",
			p95, budget, lines, len(src))
	}
}

func percentileDur(samples []time.Duration, q float64) time.Duration {
	cp := append([]time.Duration(nil), samples...)
	sort.Slice(cp, func(i, j int) bool { return cp[i] < cp[j] })
	idx := int(float64(len(cp)-1) * q)
	if idx < 0 {
		idx = 0
	}
	if idx >= len(cp) {
		idx = len(cp) - 1
	}
	return cp[idx]
}

// buildParseDoc emits a front-mattered Markdown document that exercises
// the canonical parser across its surface: headings, prose, fenced
// code, links, a table, and — the point of this package over vanilla
// goldmark — <?...?> processing-instruction blocks, so the gate covers
// the PI block parser, not just CommonMark.
func buildParseDoc(lines int) string {
	var b strings.Builder
	b.WriteString("---\ntitle: Benchmark Document\nweight: 10\n---\n\n")
	b.WriteString("# Benchmark Document\n\n")
	for i := 0; i < lines; i++ {
		switch {
		case i%25 == 0:
			fmt.Fprintf(&b, "## Section %d\n\n", i/25)
		case i%23 == 0:
			b.WriteString("<?include\nfile: other.md\nstrip-frontmatter: \"true\"\n?>\nincluded body\n<?/include?>\n\n")
		case i%17 == 0:
			b.WriteString("```go\nfunc f() int { return 0 }\n```\n\n")
		case i%13 == 0:
			b.WriteString("| Col A | Col B |\n| --- | --- |\n| a | b |\n\n")
		case i%11 == 0:
			b.WriteString("See [the spec](https://example.com/spec) for details.\n\n")
		default:
			b.WriteString("This is a synthetic sentence used to exercise " +
				"the block and inline parsers under benchmark.\n\n")
		}
	}
	return b.String()
}
