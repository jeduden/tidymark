# Conciseness Approach Matrix and Threshold Policy

This document defines the baseline approaches, threshold policy,
and model-promotion gate for conciseness experiments.

## Approach Matrix

| ID  | Family     | Core signal                            | Output |
|-----|------------|----------------------------------------|--------|
| A0  | heuristic  | current `MDS029` cues and score          | score  |
| A1  | heuristic  | tuned cue lists and weights            | score  |
| B0  | classifier | simple linear baseline                 | score  |
| B1  | classifier | selected candidate from plan 58        | score  |
| C0  | hybrid     | heuristic prefilter + classifier score | score  |

All approaches must output a numeric score in `[0, 1]`.
Higher score means higher probability of `verbose-actionable`.

## Metrics

Track these metrics on `dev`, `test`, and `holdout`:

- precision
- recall
- `F1`
- `F0.5`
- AUPRC
- Brier score
- diagnostics per KLOC
- p50 and p95 latency in milliseconds

## Threshold Policy

1. Sweep thresholds from `0.05` to `0.95` in `0.01` steps on `dev`.
2. Drop any threshold with precision below `0.75` on `dev`.
3. Pick threshold with highest `F0.5` among remaining candidates.
4. Break ties by higher recall, then lower diagnostics per KLOC.
5. Freeze one threshold per approach before running `test` and
   `holdout-outofdomain`.
6. Do not retune threshold after seeing `test` or holdout metrics.

For A0/A1 where a lower conciseness score is worse, convert with:

`risk_score = 1 - conciseness_score`

## Promotion Gate

Promote `B1` over `A0` only when all criteria pass on frozen `test`:

1. Precision improves by at least `+0.05` absolute vs `A0`.
2. Recall does not drop by more than `0.02` absolute vs `A0`.
3. `F0.5` improves by at least `+0.05` absolute vs `A0`.
4. p95 latency is less than or equal to `2x` `A0` latency.
5. Diagnostics per KLOC does not increase by more than `10%`.
6. Holdout precision is within `0.05` of test precision.

If any gate fails, keep classifier mode experimental.

## Run Artifacts

Each evaluation run must include:

- completed `scorecard-template.md`
- threshold sweep table or CSV for each approach
- confusion matrix for `test` and holdout
- benchmark environment details (CPU, Go version, dataset version)
- notes for disagreement themes and follow-up fixes
