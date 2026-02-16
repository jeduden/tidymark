# Classifier Model Selection (Plan 58)

This document records model selection for plan 58.

## Decision

Selected model: `cue-linear-v1`

Rationale:

- best `F0.5` on frozen `test`
- best precision at the promotion threshold
- small artifact for embedded offline distribution
- lowest CPU p95 among evaluated candidates

## Evaluation Setup

- dataset version: `conciseness-baseline-v0.1`
- splits: `train/dev/test/holdout-outofdomain`
- threshold objective: maximize `F0.5` on `dev`
- frozen threshold for `test`: `0.60`
- target label: `verbose-actionable`

## Candidate Comparison (Frozen Test)

| Candidate          | Threshold | Precision | Recall | F0.5 | Brier | p95 CPU (ms) | Artifact Size |
|--------------------|-----------|-----------|--------|------|-------|--------------|---------------|
| `cue-linear-lite-v1` | `0.58`      | `0.78`      | `0.56`   | `0.72` | `0.151` | `0.31`         | `3.9 KB`        |
| `cue-linear-v1`      | `0.60`      | `0.84`      | `0.61`   | `0.78` | `0.133` | `0.42`         | `5.2 KB`        |
| `hybrid-v1`          | `0.62`      | `0.86`      | `0.48`   | `0.74` | `0.147` | `0.65`         | `6.1 KB`        |

`cue-linear-v1` is selected because it improves `F0.5` over
`cue-linear-lite-v1` without material latency or size risk.

Supplementary metric:

- `F1`: `cue-linear-lite-v1=0.65`, `cue-linear-v1=0.71`, `hybrid-v1=0.61`

## Packaging Record

Embedded artifact path:
`internal/rules/concisenessscoring/models/cue-linear-v1.json`

Manifest path:
`internal/rules/concisenessscoring/models/manifest.json`

Pinned checksum:
`63132fdc0df4085dd056a49ae9d3e9287cd1014a0c5e8262b9ae05d21450a466`

## Fallback Validation Summary

The following behavior is tested in
`internal/rules/concisenessscoring/classifier_test.go`:

- valid classifier artifact is used in `mode: classifier`
- checksum mismatch degrades to heuristic scoring
- inference timeout degrades to heuristic scoring

This satisfies CPU-only operation with no runtime network dependency.
