---
diagnostics:
  - line: 3
    column: 1
    message: 'paragraph duplicated in ref/source.md:3'
---
# Duplicate Fixture

A distinctive paragraph appears in this file and in a sibling
fixture, so MDS037 must flag the match and point at the other
location. The wording stays above the default two-hundred character
threshold after normalization. It stays unique relative to the
other rule fixtures so nothing matches by accident across the test
suite.
