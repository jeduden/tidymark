// Binary resolution logic for the mdsmith extension.
// Resolves the mdsmith binary path, trying the bundled version as a
// fallback when the user-configured path is the bare "mdsmith" string.

import { existsSync } from "node:fs";
import { join } from "node:path";

// resolveBinary returns the path to the mdsmith binary. When the
// configured path is the bare string "mdsmith", it first checks
// whether the optional @mdsmith/cli dependency bundled a binary into
// node_modules/.bin/mdsmith (or the platform-specific wrapper). If
// that file exists, return its absolute path. Otherwise return the
// configured path unchanged so the LanguageClient spawns it and
// lets the shell resolve it against PATH (the original behavior).
//
// The extensionPath should be the vscode.ExtensionContext.extensionPath
// (the directory containing package.json and node_modules/).
export function resolveBinary(configuredPath: string, extensionPath: string): string {
  // If the user specified a custom path (not the bare "mdsmith"),
  // honor it exactly — they know what they want.
  if (configuredPath !== "mdsmith") {
    return configuredPath;
  }

  // The user left the default "mdsmith". Check whether the optional
  // dependency installed a bundled binary. @mdsmith/cli's bin wrapper
  // lives at node_modules/.bin/mdsmith (Unix) or
  // node_modules/.bin/mdsmith.cmd (Windows). Node package managers
  // (npm, pnpm, yarn, bun) all populate .bin/ symlinks/wrappers for
  // bin entries; we can rely on that convention.
  const binDir = join(extensionPath, "node_modules", ".bin");
  const unixBin = join(binDir, "mdsmith");
  const winBin = join(binDir, "mdsmith.cmd");

  // Prefer the platform-appropriate wrapper if it exists.
  const candidate = process.platform === "win32" ? winBin : unixBin;
  if (existsSync(candidate)) {
    return candidate;
  }

  // The bundled binary does not exist (optional dependency install
  // failed, or this is a dev build without `bun install`). Fall back
  // to the bare "mdsmith" string so the LanguageClient resolves it
  // against the shell PATH (same as before bundling).
  return configuredPath;
}
