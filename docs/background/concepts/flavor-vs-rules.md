---
summary: >-
  How "flavor" (a property of the renderer) and
  "rule" (a property of the team) differ in mdsmith,
  the cases where the two overlap, and how profiles
  bundle them for users.
---
# Flavor vs rule

mdsmith uses two related but distinct ideas to
constrain Markdown. A reader who is unsure which
mechanism they want — or whether they want both —
should read this page first.

## TL;DR

| Axis              | Flavor                             | Rule                                   |
|-------------------|------------------------------------|----------------------------------------|
| Source of truth   | A renderer (CommonMark, GFM, …)    | The team's preference                  |
| Defined by        | An external spec or implementation | An mdsmith rule package                |
| Question answered | Will the renderer interpret this?  | Does this match our convention?        |
| Diagnostic claim  | The renderer won't render this     | The team has chosen another valid form |
| Example check     | MDS034 markdown-flavor             | MDS044 horizontal-rule-style           |

## Why both exist

A flavor describes what a *renderer* does. CommonMark
won't auto-link a bare URL. GFM doesn't render
footnotes. Goldmark recognizes heading IDs but the
default GFM renderer does not. These are facts about
the parser/renderer, not opinions a team can change.

A rule describes what a *team* prefers among forms
the renderer treats equally. CommonMark accepts
`---`, `***`, and `___` as horizontal rules, with
identical output. Picking dash is a taste judgment.
The renderer doesn't care; the team does.

Without flavor, a project targeting CommonMark would
have no way to flag GFM-only syntax that will render
incorrectly on its target. Without rules, a project
on any flavor would have no way to enforce a
consistent style across files.

## The fuzzy middle

Some syntax parses on every flavor but is only
*recognized as a feature* on some. A GFM-style table
inside a CommonMark file parses fine — the renderer
emits a paragraph of literal pipes instead of an
HTML table. MDS034 still flags it, because the
renderer doesn't render the syntax as the named
feature. The diagnostic uses the wording
"<flavor> does not interpret <feature> as a feature"
to make this explicit.

Bare URLs are the cleanest case where flavor and rule
overlap legitimately. MDS034 with `flavor: commonmark`
flags them because CommonMark won't auto-link them.
MDS012 (`no-bare-urls`) flags them on style grounds
regardless of flavor. Both diagnostics fire on the
same source for different reasons. That overlap is
intentional. A team using `flavor: gfm` may still
prefer explicit `<https://example.com>` for
readability, and MDS012 enforces that even when the
renderer would auto-link.

## Profiles as the bridge

A profile is a named bundle that pairs a flavor with
a set of rule presets. `profile: portable` selects
`flavor: commonmark` and turns on a strict-style rule
set; `profile: github` selects `flavor: gfm` and
turns on a lighter set. The profile mechanism does
not collapse the two concepts. Each rule still owns
its detection logic; MDS034 still owns flavor
diagnostics. A profile only pre-fills the config
layer beneath the user's overrides.

See [profiles.md](../../reference/profiles.md) for
the built-in profiles, the merge order, and how user
overrides interact with profile presets.

## Practical guidance

Pick a flavor when the project targets a specific
renderer (GitHub, MkDocs, a static site generator,
plaintext readers). Pick rules when the team wants
to enforce a style convention. Pick a profile when
you want both at once and don't want to wire them
up by hand.

If you're seeing a diagnostic and aren't sure which
mechanism produced it, run [`mdsmith kinds resolve
<file>`](../../reference/cli.md). The per-rule merge
chain shows whether the value came from MDS034
(flavor), a style rule, or a profile preset.
