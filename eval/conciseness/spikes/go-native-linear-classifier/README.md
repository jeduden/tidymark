# Go-Native Linear Classifier Spike

## Goal

Evaluate a fully embedded pure-Go classifier path for
`verbose-actionable` detection with no runtime dynamic libraries.

## Environment

- Date: 2026-02-16
- OS: macOS 15.3.2 (arm64)
- Go: 1.24.7
- mdsmith commit: `7440e61`
- Harness: `eval/conciseness/spikes/go-native-linear-classifier/run.sh`

## Prototype

Implementation files:

- classifier package:
  `internal/rules/concisenessscoring/classifier/`
- embedded weights:
  `internal/rules/concisenessscoring/classifier/data/cue-linear-v1.json`
- benchmark command:
  `eval/conciseness/spikes/go-native-linear-classifier/main.go`

Model shape:

- sparse linear logistic model over cue rates and density metrics
- output contract:
  `label`, `risk_score`, `threshold`, `model_id`, `backend`, `version`
- runtime dependency footprint: Go stdlib only

## Classifier Implementation

Code pointers:

- loader + scorer:
  `internal/rules/concisenessscoring/classifier/model.go`
- embedded artifact:
  `internal/rules/concisenessscoring/classifier/data/cue-linear-v1.json`
- tests:
  `internal/rules/concisenessscoring/classifier/model_test.go`

Artifact schema (current spike):

```json
{
  "model_id": "cue-linear-v1",
  "version": "2026-02-16",
  "threshold": 0.20,
  "intercept": -0.85,
  "weights": {
    "action_rate": -2.40,
    "content_ratio": -2.60,
    "filler_rate": 6.10,
    "hedge_rate": 7.40,
    "log_word_count": 0.18,
    "modal_rate": 5.20,
    "vague_rate": 4.60,
    "verbose_phrase_rate": 8.80
  }
}
```

Scoring equation:

```text
score = intercept + Σ(weight_i * feature_i)
risk_score = sigmoid(score)
label = "verbose-actionable" if risk_score >= threshold else "acceptable"
```

Feature extraction in `model.go`:

- tokenized lowercase words via regex (`[a-z0-9']+`)
- normalized rates per word:
  `filler_rate`, `hedge_rate`, `verbose_phrase_rate`, `modal_rate`,
  `vague_rate`, `action_rate`
- density/length terms:
  `content_ratio`, `log_word_count`

Determinism and safety properties:

- no network calls and no external dynamic libraries
- `go:embed` + pinned SHA256 verification on load
- same input string always yields the same `risk_score` and `label`

## Embedded Weight Packaging

Artifact details:

- path: `data/cue-linear-v1.json`
- embedded bytes: 335
- SHA256:
  `a17544b94507ad05e5d9db33078ca7a63d3fccd94b1d091ee1c85a88bbc81e44`

Load path:

1. embed artifact bytes via `go:embed`
2. verify SHA256 against pinned constant
3. decode JSON into model weights and threshold
4. fail closed on checksum/decode/validation errors

## Corpus and Determinism

The benchmark corpus matches the six-sample set used in the yzma spike
(`weasel-01..03` and `direct-01..03`).

Determinism checks:

- in-process repeats (`determinism-runs=5`): `unique_hashes=1`
- process-restart repeats (`run.sh`): `unique_hashes=1`
- digest:
  `41f7fe0f2d6d755b647f4f923f79c5d682cfcda75add60a7e2df3fcba29fce08`

Result: deterministic outputs were confirmed.

## Latency and Memory

Measured with `ROUNDS=4000` (`24,000` requests total):

| Metric                 | Value       |
|------------------------|-------------|
| Model load             | 0.0310 ms   |
| Avg latency            | 2.9299 us   |
| P50 latency            | 2.8750 us   |
| P95 latency            | 3.6670 us   |
| Max latency            | 230.7910 us |
| RSS after load         | 3,872 KB    |
| RSS after benchmark    | 7,744 KB    |
| Heap alloc after bench | 421 KB      |
| Heap sys after bench   | 7,744 KB    |

Per-sample risk scores:

- `weasel-01`: 0.6323
- `weasel-02`: 0.3833
- `weasel-03`: 0.2101
- `direct-01`: 0.0472
- `direct-02`: 0.0619
- `direct-03`: 0.0627

The embedded threshold for this spike artifact is `0.20`.

## Captured Run Output

Raw `run.sh` benchmark output captured on 2026-02-16:

```text
model_id=cue-linear-v1
model_version=2026-02-16
threshold=0.2000
artifact_path=data/cue-linear-v1.json
artifact_sha256=a17544b94507ad05e5d9db33078ca7a63d3fccd94b1d091ee1c85a88bbc81e44
artifact_bytes=335
model_load_ms=0.0310
rss_after_load_kb=3872
determinism_digest=41f7fe0f2d6d755b647f4f923f79c5d682cfcda75add60a7e2df3fcba29fce08
determinism_unique_hashes=1
requests=24000
avg_latency_us=2.9299
p50_latency_us=2.8750
p95_latency_us=3.6670
max_latency_us=230.7910
rss_after_bench_kb=7744
heap_alloc_after_bench_kb=421
heap_sys_after_bench_kb=7744
total_alloc_delta_kb=31005
labels_verbose_actionable=12000
labels_acceptable=12000
risk_direct-01=0.0472
risk_direct-02=0.0619
risk_direct-03=0.0627
risk_weasel-01=0.6323
risk_weasel-02=0.3833
risk_weasel-03=0.2101
```

Raw binary-size output from the same run:

```text
mdsmith_base_bytes=24511378
mdsmith_go_native_bytes=24512514
mdsmith_delta_bytes=1136
```

## Binary-Size Impact

Measured by building `cmd/mdsmith` with and without the
`spike_gonative_classifier` tag:

- base mdsmith: 24,511,378 bytes
- mdsmith with embedded classifier artifact: 24,512,514 bytes
- delta: 1,136 bytes

Comparison with yzma spike footprint:

- go-native spike: +1.1 KB binary delta, no external model/libs
- yzma spike: +0.5 MB binary delta plus 84 MB model and 86 MB libs

## MDS029 Integration Boundaries

Recommended runtime modes:

- `classifier`: force embedded classifier path
- `heuristic`: force current MDS029 heuristic path
- `auto`: try classifier first, degrade to heuristic on failure

Recommended timeout policy:

- classify with bounded context timeout (for example 2 ms default)
- on timeout, treat as classifier failure and fall back to heuristic
- count fallback events for `--verbose` reporting only

Diagnostic stability requirements:

- preserve one diagnostic schema across backends
- keep thresholded binary decision semantics stable
- include `backend` and `model_id` in verbose/debug output only

## Maintenance Workflow

Safe model update path:

1. train/export next artifact in deterministic JSON format
2. replace `cue-linear-v1.json` with versioned artifact content
3. recompute SHA256 and update pinned checksum constant
4. run `go test ./internal/rules/concisenessscoring/classifier`
5. run `eval/conciseness/spikes/go-native-linear-classifier/run.sh`
6. record version, checksum, and size/latency deltas in release notes

Versioning guidance:

- keep explicit artifact version in JSON (`version`)
- keep model identifier stable (`model_id`) until feature contract changes
- require checksum updates in the same commit as weight updates

## Recommendation

Adopt this path as the default CPU fallback candidate for plan 58.

Rationale:

- deterministic behavior is straightforward to guarantee
- binary and artifact footprint are effectively negligible
- runtime has no dynamic-library or external-service dependency

Remaining gap:

- quality must still be validated on the larger plan-59 dataset before
  promoting classifier mode to default for MDS029.
