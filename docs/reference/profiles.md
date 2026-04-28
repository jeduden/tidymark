---
summary: >-
  Built-in markdown-flavor profiles, the rule presets
  each one applies, and how user config layers on top
  via deep-merge.
---
# Markdown-flavor profiles

A **profile** is an opinionated bundle of rule
settings that pairs a Markdown flavor with a set of
style choices. Setting `profile:` in your
`markdown-flavor` config selects one of the built-in
bundles; the rule presets in that bundle are applied
as a base layer beneath your own rule config.

Profiles answer "what subset of Markdown does this
project use?" with one config knob instead of eight.

Profiles bridge two distinct ideas in mdsmith. A
flavor is a property of the *renderer*; a rule is a
property of the *team*. See
[flavor vs rule](../background/concepts/flavor-vs-rules.md)
for the conceptual split and where the two overlap.

## Selecting a profile

```yaml
rules:
  markdown-flavor:
    profile: portable
```

`profile:` accepts one of `portable`, `github`, or
`plain`. Setting an unknown profile name is a config
error at load time.

You may also set `flavor:` alongside `profile:`. If
both are set, they must agree — a profile that
requires `commonmark` rejects `flavor: gfm` at config
load.

## Built-in profiles

### `portable`

Markdown that renders the same in every CommonMark
parser. Selects `flavor: commonmark` and turns on the
strict-style rules with their recommended defaults.

| Rule                     | Setting                                                 |
|--------------------------|---------------------------------------------------------|
| `markdown-flavor`        | `flavor: commonmark`                                    |
| `no-inline-html`         | enabled                                                 |
| `no-reference-style`     | `allow-footnotes: false`                                |
| `emphasis-style`         | `bold: asterisk`, `italic: underscore`                  |
| `horizontal-rule-style`  | `style: dash`, `length: 3`, `require-blank-lines: true` |
| `list-marker-style`      | `style: dash`                                           |
| `ordered-list-numbering` | `style: sequential`, `start: 1`                         |
| `ambiguous-emphasis`     | `max-run: 2`                                            |

### `github`

Markdown that renders well on github.com. Selects
`flavor: gfm` and keeps the style rules light: the
inline-HTML allowlist permits `<details>` and
`<summary>`; emphasis and list-marker style are
pinned for consistency; the rest of the strict rules
stay off.

| Rule                | Setting                                |
|---------------------|----------------------------------------|
| `markdown-flavor`   | `flavor: gfm`                          |
| `no-inline-html`    | `allow: [details, summary]`            |
| `emphasis-style`    | `bold: asterisk`, `italic: underscore` |
| `list-marker-style` | `style: dash`                          |

### `plain`

Markdown that survives `cat`. The rendered output
should look about the same as the source viewed in a
plaintext reader. Same activations as `portable`,
plus `allow-comments: false` on `no-inline-html` so
HTML comments do not leak through as literal `<!--
... -->` text.

A truly plaintext-faithful profile needs three more
rules. One forbids `*` and `_` runs. One requires
indented code blocks. One inverts `no-bare-urls` so
bare URLs are preferred over Markdown links. Those
rules don't exist yet. When they ship, the `plain`
profile gains them and diverges from `portable`.

## How presets layer with user config

Profile presets are a **base layer** beneath your
top-level `rules:` block. The merge order, oldest →
newest, is:

1. `profile.<name>` — the preset table
2. `default` — built-in defaults plus your top-level rules
3. `kinds.<name>` — each kind in the file's effective list
4. `overrides[i]` — each matching override entry

Each layer deep-merges onto the previous one. Scalars
at a leaf are replaced by the later layer; maps
recurse key by key; lists replace by default. So a
profile preset provides the floor, and your config
overrides on top.

For example, the `github` profile sets
`no-inline-html.allow: [details, summary]`. To extend
the allowlist with `<sub>` and `<sup>`, write:

```yaml
rules:
  markdown-flavor:
    profile: github
  no-inline-html:
    allow: [sub, sup]
```

Lists default to replace, so the effective allowlist
becomes `[sub, sup]`. To keep the preset's entries,
list them explicitly:
`allow: [details, summary, sub, sup]`.

## Disabling MDS034

A profile applies its rule presets at config load
time, so disabling `markdown-flavor` itself does not
disable the rules a profile turned on:

```yaml
rules:
  markdown-flavor:
    profile: portable
    # …then disable MDS034's own diagnostics:
  markdown-flavor: false
```

The other rules' presets stay in effect. Only
MDS034's own flavor reporting is silenced. The split
keeps MDS034 focused on "what does this renderer
support." Profiles orchestrate style separately.

## Inspecting an effective profile

`mdsmith kinds resolve <file>` shows the merge chain
for every rule, including the `profile.<name>` layer.
Use it to confirm which value won and where it came
from.
