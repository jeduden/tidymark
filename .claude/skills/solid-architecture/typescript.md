---
title: TypeScript SOLID and clean architecture patterns
summary: >-
  Language-level SOLID and clean architecture
  rules for TypeScript codebases, with a bias
  toward VS Code extension shapes.
  Project-agnostic.
---
# TypeScript SOLID and clean architecture patterns

Language-level rules. The project's own
docs describe its actual module names and
layout; this file describes the patterns
that should hold regardless of project.

## Layout

A small extension or library should stay
small. A defensible target shape:

```text
src/
├── extension.ts        (entry; thin)
├── wiring.ts           (compose
│                        dependencies)
├── <subsystem>.ts      (e.g. binary
│                        location)
└── commands/
    ├── <command>.ts
    └── runner.ts       (typed exec /
                         shared I/O)
```

Tests live next to source as `*.test.ts`.

## Single responsibility per module

A module exports one cohesive answer. A
`commands/<name>.ts` exports the command's
registration; the shared exec layer lives
in one module that every command consumes.
Do not copy the shared work into each
command.

A module past ~200 lines of behavior
should split. The natural split is three
modules: one decides what to run, one
runs it, one renders the result.

## Open/closed: adding a feature

Adding a new command, a new menu item, or
a new editor action should not modify the
entry file. Steps:

1. Create the new module with a
   registration function.
2. Wire it from the composition module.
3. Add the matching entry to
   `package.json` `contributes` (or the
   relevant manifest).
4. Add a test next to the module.

The contributes section is part of the
extension's public contract; changing the
id or arguments of a shipped command is a
breaking change.

## Liskov: registration shapes are interchangeable

Every command takes the same wiring
shape: a typed dependency object or the
host's `ExtensionContext`. Every command
returns `Promise<void>` or a
`Disposable`. Do not introduce optional,
command-specific parameters at the
registration site. If a command needs
extra configuration, read it from
workspace settings inside the command.

A command must commit to one return
shape. "Maybe done, maybe not" violates
Liskov. The caller should not need to
know which kind it received.

## Interface segregation at typed boundaries

The host API surface is wide; consumers
should depend only on the slice they use.
Define narrow shape interfaces alongside
each consumer:

```ts
interface FileSystemWatcherLike {
  onDidChange(listener: (uri: Uri) => void): Disposable;
  dispose(): void;
}
```

A small typed surface lets the consumer
test against a fake without reaching for
a full mock of the host.

## Dependency inversion through typed boundaries

An extension typically communicates with
the host and with subprocesses. Each
boundary is a typed contract:

- **Wire protocols** (LSP, RPC) — typed
  via the protocol library. Capabilities
  are the contract.
- **Subprocess invocation** — typed by
  the schema returned by each subcommand,
  parsed once at the boundary into a
  domain shape.
- **File system reads** — typed via
  narrow shape interfaces (e.g. a parsed
  config shape, not raw YAML or JSON).

The boundary between strings and types
should be exactly one function deep. Do
not thread unparsed values through the
rest of the codebase.

## Clean architecture in the entry file

Target: the entry file is wiring only:

- Activate the extension.
- Construct the wiring object.
- Hand control to a composition module.

The entry file should not own state, not
register commands directly, and not
construct client objects whose lifecycle
extends beyond activation. If it does any
of those today, that is fine — but new
code should be placed where the target
shape expects it, so the entry file does
not grow further.

## Testing patterns

- Pure unit tests for each module using
  the project's test runner.
- Extract pure functions out of command
  bodies and unit-test those instead of
  mocking the host API. Mock the host
  only when unavoidable.
- For subprocess boundaries, prefer
  running the real binary against a
  fixture workspace.
- Place test fixtures near the test that
  uses them. Do not import fixture data
  across command modules.

## Common violations to flag

- A command that imports another command.
- A command that constructs paths to
  another command's artifacts; share
  state through the composition module.
- A type declared in the entry file that
  is used by commands — should live in
  the composition module or its own
  module.
- Raw `child_process.exec` outside the
  designated subprocess module.
- A field added to `package.json`
  `contributes` without a corresponding
  module that owns it.
- A test that spins up the full host to
  test logic that could be unit-tested
  out of band.
- A registration call inlined in the
  activation body rather than routed
  through the registration helper.
- A `Util`, `Helpers`, or `Misc` module
  anywhere in `src/`.

## Refactor moves that usually work

- A command with three responsibilities
  splits into three commands or one
  command plus two helpers in the shared
  runner module.
- A growing composition module: extract
  a registry module that commands enroll
  into; the composition module becomes a
  thin call list.
- Raw JSON crossing module boundaries:
  parse into a typed shape at the
  subprocess boundary and pass the shape
  onward.
- Scattered `vscode.workspace.getConfiguration`
  calls: centralize into a config-reader
  module typed to the project's settings
  schema.
