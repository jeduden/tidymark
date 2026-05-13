---
title: Cross-system contract patterns
summary: >-
  Language-agnostic patterns for the
  external surfaces a project ships: wire
  protocols, CLI flags, config schemas,
  distribution shims, plugin manifests.
  Project-agnostic.
---
# Cross-system contract patterns

A project that ships multiple surfaces
binds itself to contracts. Those contracts
are more expensive to change than internal
code. Consumers ship and release on their
own schedule.

Each surface is a public API. This file
covers the patterns that govern them; the
project's own docs name the specific
surfaces and their owner packages.

## Typical surfaces

| Surface                | What it is                                                        |
|------------------------|-------------------------------------------------------------------|
| Wire protocol          | LSP, gRPC, or similar — typed capabilities a client subscribes to |
| CLI flags + exit codes | The shell-script-facing contract                                  |
| Config schema          | The user-facing structured input (YAML, TOML, JSON)               |
| Generated markers      | In-file placeholders the tool reads back                          |
| Plugin manifest        | Editor/IDE manifest declaring servers, commands, hooks            |
| Marketplace listing    | The discovery surface for a plugin                                |
| Distribution shim      | Language-specific install wrapper (npm, pip, …)                   |
| Editor `contributes`   | The plugin's contributed commands and settings                    |

A breaking change at any of these surfaces
is a SemVer-major event.

## Dependency inversion at the boundary

The binary (or library) is the source of
truth. Every external surface adapts to
it; it does not adapt to them.

- **Wire protocol**: the server exposes
  capabilities; clients subscribe. The
  server does not branch on which client
  connected.
- **Distribution shims**: exec the binary
  with forwarded args. They never inspect
  domain data or duplicate rule logic.
- **Plugin manifest**: declares the
  server invocation; it does not embed
  domain logic.
- **Editor extension**: consumes the
  server and the binary. It does not
  implement domain rules.

If a shim must translate something — a
platform-specific binary download, a
fallback to a checked-in copy, a JSON
re-shape — that translation lives in the
shim and is unit-tested there. It is not
allowed to reach into the binary's
internals.

## Interface segregation across surfaces

A consumer of one surface should not be
forced to install another. The npm shim
consumer should not need the LSP server
binary; the editor extension consumer
should not need the standalone plugin.
Keep each surface narrow and
self-contained; package-level optional
dependencies should reflect that.

When the binary gains a feature, ask
which surface(s) need to expose it. Only
the ones that benefit. Resist exposing
new fields on every JSON envelope "in
case someone wants them".

## Versioning and compatibility

- The **CLI** follows SemVer. Adding a
  flag is minor; renaming or removing
  one is major. Behavior changes that
  turn a passing run into a failing run
  are major.
- **Wire-protocol capabilities** can be
  additive without bumping major.
  Removing a capability is major.
- **Config schemas** are additive within
  a major. Removing a field requires a
  deprecation window or a major bump.
  Renaming a field is removing one and
  adding another.
- **Generated markers** are forever.
  Once a marker syntax is shipped, the
  binary must continue to parse it
  (possibly with a deprecation
  diagnostic) until the next major.
- **Plugin manifest** schemas follow the
  host's versioning. Treat the set of
  declared servers, commands, and hooks
  as the visible interface.

## Liskov across distribution shims

Every shim is a substitute for invoking
the binary directly. The shim and the
binary must behave identically:

- Same exit codes.
- Same stdout / stderr formats.
- Same flag parsing.

If a shim deviates (e.g. adds a flag the
binary does not have), that is a Liskov
violation. Push the flag down to the
binary or drop it from the shim.

## Anti-patterns

- A shim that parses domain data to
  decide which args to pass.
- An editor command that reads the
  config file directly instead of asking
  the binary or the wire-protocol server.
- A plugin command that wraps a shell
  call with bespoke logic — promote the
  logic into a binary subcommand and
  call that.
- A field added to the config schema for
  a feature that should be a CLI flag
  (or vice versa). Rule of thumb:
  file-wide configuration lives in the
  schema; per-run overrides live on the
  CLI.
- A diagnostic or response shape that
  varies by output mode. The shape is
  the contract; if a mode wants
  different data, define a different
  mode.
- A breaking change to a shipped CLI
  flag or schema field with no
  deprecation window.
- A new wire-protocol capability the
  shipped client is expected to use but
  does not yet opt into.

## Reviewing a new surface

When adding a new integration — a new
editor, a new distribution channel, a
new plugin host:

1. Identify which existing surface(s) it
   composes with. If none, ask whether
   the new surface really needs to exist
   or could be served by an existing one.
2. Write down the contract — what data
   flows in and out, in what format.
   Store it in the project's reference
   docs.
3. Add a test that locks the contract
   inside the binary (a CLI flag test, a
   protocol fixture, etc.). Without a
   contract test, the surface drifts
   under refactor pressure.
4. Decide the versioning policy.
5. Confirm the surface follows the
   dependency-inversion rule: it adapts
   to the binary; the binary does not
   adapt to it.

## Refactor moves that usually work

- Duplicate logic across shims → push
  the logic into a binary subcommand
  consumed by all shims.
- A field that lives in two contracts
  (e.g. CLI + config) and means
  different things → split into two
  fields with distinct names.
- A capability that exists only in one
  editor → either generalize it to the
  wire protocol or scope it to that
  editor's package, never both
  half-formed.
- A test that touches three contracts →
  split into three tests, one per
  contract.
