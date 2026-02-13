---
title: Choosing Readability vs Conciseness Metrics
description: Trade-offs, examples, and threshold guidance for readability, structure, length, and conciseness rules.
---
# Choosing Readability vs Conciseness Metrics

## Scope and disclaimer

This guide compares existing mdsmith rules that touch readability and length with the proposed conciseness scoring idea. Conciseness scoring is not implemented yet and still needs design; any conciseness numbers below are illustrative, not normative.

## What the current rules measure

| Rule | Measures | Default | What it misses |
|------|----------|---------|----------------|
| [MDS023](../rules/MDS023-paragraph-readability/README.md) `paragraph-readability` | Complexity using ARI (characters per word, words per sentence) | `max-grade: 14.0`, `min-words: 20` | Wordiness and filler; short but dense paragraphs can be skipped |
| [MDS024](../rules/MDS024-paragraph-structure/README.md) `paragraph-structure` | Shape and length of paragraphs (sentences per paragraph, words per sentence) | `max-sentences: 6`, `max-words: 40` | Verbosity that fits within limits; dense but short prose |
| [MDS022](../rules/MDS022-max-file-length/README.md) `max-file-length` | Lines per file | `max: 300` | Token load and dense paragraphs |
| [MDS001](../rules/MDS001-line-length/README.md) `line-length` | Characters per line | `max: 80` | Verbosity and paragraph complexity |

## What conciseness scoring is trying to measure

Conciseness scoring focuses on information density rather than complexity or structure. It aims to flag paragraphs that are easy to read but say too little with too many words, which can waste tokens and create drift in agentic contexts. A plausible starting point is a heuristic that penalizes filler words, hedging language, and verbose phrases while rewarding content-bearing terms.

## Example paragraphs

### Example A

In order to make sure that we are all on the same page, it is important to note that the system is, in most cases, able to handle requests pretty well, and this is something we should keep in mind.

### Example B

The synchronization algorithm enforces linearizability via per-shard lease epochs and monotonic commit indices.

### Example C

We should update the onboarding guide so that new contributors can quickly find the build steps, understand the release checklist, and avoid common pitfalls without needing to ask in chat, which will reduce interruptions for everyone.

### Example D

The plan is straightforward. We will add a new rule. It will report issues. It will include guidance. It will ship this week. It will help teams. It will reduce noise. It will keep docs short.

### Example E

Basically, we just want to make sure the plan is pretty clear to everyone. It is really just a simple update, and we might adjust it later.

## How the rules score these examples

Notes: ARI values use mdsmith's current formula. [MDS023](../rules/MDS023-paragraph-readability/README.md) skips paragraphs under `min-words`. [MDS024](../rules/MDS024-paragraph-structure/README.md) flags when sentences or words exceed limits. Conciseness scores below are illustrative heuristics, not an implemented rule.

| Example | Words | Sentences | ARI | MDS023 result | MDS024 result | Conciseness score (illustrative) |
|---------|------:|----------:|----:|--------------|---------------|----------------------------------|
| A | 40 | 1 | 16.6 | Fail (16.6 > 14.0) | Pass | 36.2 |
| B | 13 | 1 | 20.2 | Skipped (< 20 words) | Pass | 84.6 |
| C | 36 | 1 | 22.1 | Fail (22.1 > 14.0) | Pass | 63.9 |
| D | 36 | 8 | 0.3 | Pass | Fail (8 > 6 sentences) | 50.0 |
| E | 26 | 2 | 4.6 | Pass | Pass | 50.0 |

## Trade-offs by metric

| Metric | Strengths | Risks |
|--------|-----------|-------|
| Readability ([MDS023](../rules/MDS023-paragraph-readability/README.md)) | Encourages simple, broadly accessible prose | Penalizes technical terms; misses wordiness; can skip short dense paragraphs |
| Structure ([MDS024](../rules/MDS024-paragraph-structure/README.md)) | Enforces consistent paragraph shape with low false positives | Does not address filler or redundancy |
| Length ([MDS022](../rules/MDS022-max-file-length/README.md), [MDS001](../rules/MDS001-line-length/README.md)) | Prevents runaway size and formatting drift | Poor proxy for token load or verbosity |
| Conciseness (proposed) | Targets verbosity and token waste | Heuristic; can penalize necessary qualifiers or legal language |

## How to choose limits

1. Start with defaults for [MDS023](../rules/MDS023-paragraph-readability/README.md) and [MDS024](../rules/MDS024-paragraph-structure/README.md) to establish baseline structure and readability.
2. Sample a representative set of documents and collect results before tightening thresholds.
3. For conciseness scoring, set an initial threshold that flags only the worst 10 to 20 percent of paragraphs, then adjust.
4. Use path-based overrides to reflect different document types, such as onboarding guides vs architecture specs.
5. Re-evaluate thresholds after major content changes or when onboarding new teams.

## When to use one measure instead of many

If you need a single metric to minimize complexity, choose the one that best matches your risk:

- Choose [MDS024](../rules/MDS024-paragraph-structure/README.md) `paragraph-structure` when you want predictable, low-noise enforcement.
- Choose [MDS023](../rules/MDS023-paragraph-readability/README.md) `paragraph-readability` when broad comprehension is the highest priority.
- Choose conciseness scoring when token budget and drift are the main risks and you accept heuristic trade-offs.

## Recommendation for mdsmith users

Start with [MDS023](../rules/MDS023-paragraph-readability/README.md) and [MDS024](../rules/MDS024-paragraph-structure/README.md) enabled. Use [MDS022](../rules/MDS022-max-file-length/README.md) and [MDS001](../rules/MDS001-line-length/README.md) as baseline file and line controls. Add conciseness scoring only after calibrating its thresholds and confirming it improves signal without harming necessary precision.
