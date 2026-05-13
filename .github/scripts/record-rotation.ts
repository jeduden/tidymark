#!/usr/bin/env bun
/**
 * Update the `last-rotated` date for an entry in the front matter
 * of `docs/development/secret-rotations.md`. In place.
 *
 * Invoked from `.github/workflows/record-secret-rotation.yml` after
 * the maintainer has rotated a credential at its issuer. The
 * workflow then commits the result on a fresh branch and opens a
 * PR so CODEOWNERS and the mdsmith lint job both gate the change.
 *
 * Usage:
 *     bun run record-rotation.ts <ENTRY_NAME> <YYYY-MM-DD>
 *
 * The argument is the entry's name in the rotations table (e.g.
 * `VSCE_PAT`), never the credential value. The script never reads,
 * prints, or stores any credential material — `entryName` flows
 * only as a lookup key.
 *
 * Exit codes:
 * - 0: success — front matter updated, or the date was already set
 *   to the requested value (no-op)
 * - 1: doc malformed, name not found, or date not valid ISO-8601
 */

import { resolve } from "node:path";
import { parse as yamlParse } from "yaml";

const REPO_ROOT = resolve(import.meta.dir, "..", "..");
const ROTATION_DOC = resolve(REPO_ROOT, "docs/development/secret-rotations.md");

class SystemExit extends Error {}

interface SplitFrontMatter {
  /** The literal "---\n" opener. */
  opening: string;
  /** The YAML between the two "---\n" fences, fences NOT included. */
  yamlBlock: string;
  /** The closing "\n---\n" fence plus the rest of the document. */
  closingPlusBody: string;
}

/** Split front matter from a markdown source. Concatenating the
 * three pieces reproduces the input byte-for-byte. */
function splitFrontMatter(text: string): SplitFrontMatter {
  if (!text.startsWith("---\n")) {
    throw new SystemExit(`${ROTATION_DOC}: no front matter`);
  }
  const end = text.indexOf("\n---\n", 4);
  if (end === -1) {
    throw new SystemExit(`${ROTATION_DOC}: unterminated front matter`);
  }
  return {
    opening: text.slice(0, 4),
    yamlBlock: text.slice(4, end),
    closingPlusBody: text.slice(end),
  };
}

function escapeRegex(s: string): string {
  return s.replace(/[.*+?^${}()|[\]\\]/g, "\\$&");
}

/** Rewrite the YAML to set last-rotated for `entryName` to `date`.
 *
 * The structural rewrite uses a YAML parse to confirm the entry
 * exists. The actual textual edit uses a regex on the source so
 * unrelated formatting (key order, quoting style, comments) is
 * preserved.
 *
 * The matcher tolerates indented sibling keys AND blank lines
 * between the `- name:` and `last-rotated:` keys. It does NOT
 * handle `last-rotated:` appearing BEFORE `name:` in the same
 * entry — reordering would also break check-secret-rotations.ts's
 * identity-by-name lookup, so canonical ordering is enforced by
 * convention elsewhere.
 */
function updateLastRotated(yamlBlock: string, entryName: string, date: string): string {
  const parsed = yamlParse(yamlBlock);
  if (parsed === null || typeof parsed !== "object" || Array.isArray(parsed)) {
    throw new SystemExit("front matter is not a mapping");
  }
  const rotations = (parsed as { rotations?: unknown }).rotations;
  if (!Array.isArray(rotations)) {
    throw new SystemExit("front matter has no `rotations:` list");
  }
  const names: string[] = [];
  let found = false;
  for (const r of rotations) {
    if (r !== null && typeof r === "object" && !Array.isArray(r)) {
      const name = (r as { name?: unknown }).name;
      if (typeof name === "string") {
        names.push(name);
        if (name === entryName) found = true;
      }
    }
  }
  if (!found) {
    throw new SystemExit(`unknown name '${entryName}'; known: ${names.join(", ")}`);
  }

  // Captures:
  //   group(1): the `- name: NAME\n` line + indented preamble +
  //             the `last-rotated:` key
  //   group(2): the current value of last-rotated
  const pattern = new RegExp(
    `(- name:\\s*${escapeRegex(entryName)}\\b[^\n]*\n` +
      `(?:[ \\t]+[^\n]*\n|[ \\t]*\n)*?` +
      `[ \\t]+last-rotated:\\s*)` +
      `("?[^"\n]*"?)`,
  );
  let matched = 0;
  const rewritten = yamlBlock.replace(pattern, (_, preamble) => {
    matched++;
    return `${preamble}"${date}"`;
  });
  if (matched === 0) {
    throw new SystemExit(
      `could not locate \`last-rotated:\` line under \`- name: ${entryName}\``,
    );
  }
  return rewritten;
}

function isIsoDate(s: string): boolean {
  if (!/^\d{4}-\d{2}-\d{2}$/.test(s)) return false;
  const parsed = new Date(`${s}T00:00:00Z`);
  return !Number.isNaN(parsed.getTime());
}

async function main(argv: string[]): Promise<number> {
  if (argv.length !== 4) {
    process.stderr.write("usage: bun run record-rotation.ts <ENTRY_NAME> <YYYY-MM-DD>\n");
    return 1;
  }
  const entryName = argv[2]!;
  const dateStr = argv[3]!;
  if (!isIsoDate(dateStr)) {
    process.stderr.write(`invalid date '${dateStr}': not a valid ISO-8601 date\n`);
    return 1;
  }
  const text = await Bun.file(ROTATION_DOC).text();
  const { opening, yamlBlock, closingPlusBody } = splitFrontMatter(text);
  const updated = updateLastRotated(yamlBlock, entryName, dateStr);
  if (updated === yamlBlock) {
    console.log(`${entryName} already at ${dateStr}; no change`);
    return 0;
  }
  await Bun.write(ROTATION_DOC, opening + updated + closingPlusBody);
  console.log(`updated ${entryName}.last-rotated -> ${dateStr}`);
  return 0;
}

try {
  process.exit(await main(Bun.argv));
} catch (err) {
  if (err instanceof SystemExit) {
    process.stderr.write(`${err.message}\n`);
    process.exit(1);
  }
  throw err;
}
