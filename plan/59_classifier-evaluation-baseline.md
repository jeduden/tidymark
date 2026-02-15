---
id: 59
title: Classifier Evaluation Baseline
status: ✅
---
# Classifier Evaluation Baseline

## Goal

Set up a clear baseline to test conciseness methods.
Use shared data and metrics to compare heuristic,
classifier, and hybrid results.

## Baseline Assets

Baseline documents are checked in under `eval/conciseness/`:

- `rubric.md`: inclusion/exclusion policy with canonical examples
- `dataset.schema.cue`: JSONL record schema and cue taxonomy
- `approach-matrix.md`: A0/A1/B0/B1/C0 matrix and threshold policy
- `scorecard-template.md`: dev/test metrics, disagreement,
  shadow-trial outcomes, and decision gate table
- `README.md`: dataset split policy and sampling requirements

## Threshold and Gate Policy

Thresholds are tuned on `dev` only and frozen before `test`.
Primary objective is `F0.5` with a precision floor.
Promotion gates require quality improvement over A0 with
bounded latency and diagnostics overhead.

## Tasks

1. Define annotation rubric and label policy for
   `verbose-actionable` vs `acceptable`.
2. Create benchmark dataset format, split policy, and baseline
   sampling requirements across document types.
3. Implement an experiment matrix that compares at least:
   A0 (MDS029 heuristic), A1 (heuristic + lexicon tuning),
   B0 (simple classifier baseline), B1 (PR #15 classifier),
   and C0 (hybrid).
4. Define evaluation metrics and decision thresholds:
   precision, recall, F0.5, AUPRC, calibration,
   diagnostics per KLOC, and latency overhead.
5. Add reporting templates for dev/test metrics,
   disagreement review, and shadow-trial outcomes.

## Acceptance Criteria

- [x] A written rubric exists with inclusion/exclusion guidance
      and at least 10 canonical examples.
- [x] Dataset schema and split policy are documented and checked in.
- [x] The approach matrix and threshold policy are documented.
- [x] A scorecard template exists for per-approach comparison.
- [x] Decision gate criteria are defined for promoting a model.
- [x] `mdsmith check .` reports no issues.
