---
settings:
  max: 0
  per-heading:
    - pattern: "^Short$"
      max: 4
---
# Title

## Short

One line.

## Longer section with no matching pattern

This section is longer but stays untouched because only headings
matching `^Short$` are capped and the default `max` is zero.
