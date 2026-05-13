import { describe, expect, test } from "bun:test";
import {
  isIsoDate,
  splitFrontMatter,
  SystemExit,
  updateLastRotated,
} from "./record-rotation";

describe("isIsoDate", () => {
  test("accepts a real calendar date", () => {
    expect(isIsoDate("2026-05-13")).toBe(true);
  });
  test("rejects a calendar-invalid date that Date.parse normalizes", () => {
    expect(isIsoDate("2026-02-31")).toBe(false);
  });
  test("rejects malformed shapes", () => {
    expect(isIsoDate("2026-5-13")).toBe(false);
    expect(isIsoDate("not-a-date")).toBe(false);
    expect(isIsoDate("")).toBe(false);
  });
});

describe("splitFrontMatter", () => {
  test("splits a well-formed file into opening / yaml / closing+body", () => {
    const text = '---\ntitle: VSCE_PAT\nperiodDays: 335\n---\n# body\n';
    const out = splitFrontMatter(text, "test.md");
    expect(out.opening).toBe("---\n");
    expect(out.yamlBlock).toBe("title: VSCE_PAT\nperiodDays: 335");
    expect(out.closingPlusBody).toBe("\n---\n# body\n");
    // Round-trips: concatenation recovers the original.
    expect(out.opening + out.yamlBlock + out.closingPlusBody).toBe(text);
  });
  test("rejects a file with no front matter", () => {
    expect(() => splitFrontMatter("no front matter here\n", "test.md"))
      .toThrow(SystemExit);
  });
  test("rejects an unterminated front-matter block", () => {
    expect(() => splitFrontMatter("---\ntitle: X\nno close fence\n", "test.md"))
      .toThrow(SystemExit);
  });
});

describe("updateLastRotated", () => {
  test("rewrites a bare unquoted value", () => {
    const block = 'title: VSCE_PAT\nlastRotated: 2026-04-01\nperiodDays: 335';
    const out = updateLastRotated(block, "2026-05-12", "vsce-pat.md");
    expect(out).toBe('title: VSCE_PAT\nlastRotated: "2026-05-12"\nperiodDays: 335');
  });
  test("rewrites a double-quoted value", () => {
    const block = 'lastRotated: "2026-04-01"\nperiodDays: 335';
    const out = updateLastRotated(block, "2026-05-12", "vsce-pat.md");
    expect(out).toBe('lastRotated: "2026-05-12"\nperiodDays: 335');
  });
  test("rewrites a single-quoted value and normalizes to double quotes", () => {
    const block = "lastRotated: '2026-04-01'\nperiodDays: 335";
    const out = updateLastRotated(block, "2026-05-12", "vsce-pat.md");
    expect(out).toBe('lastRotated: "2026-05-12"\nperiodDays: 335');
  });
  test("preserves a trailing inline comment after the value", () => {
    const block = 'lastRotated: 2026-04-01 # rotated after incident\nperiodDays: 335';
    const out = updateLastRotated(block, "2026-05-12", "vsce-pat.md");
    expect(out).toBe('lastRotated: "2026-05-12" # rotated after incident\nperiodDays: 335');
  });
  test("tolerates leading indentation on the key", () => {
    const block = '  lastRotated: 2026-04-01\n  periodDays: 335';
    const out = updateLastRotated(block, "2026-05-12", "vsce-pat.md");
    expect(out).toBe('  lastRotated: "2026-05-12"\n  periodDays: 335');
  });
  test("throws when the lastRotated line is absent", () => {
    expect(() => updateLastRotated("title: X\nperiodDays: 30", "2026-05-12", "x.md"))
      .toThrow(SystemExit);
  });
  test("returns identical bytes when the date is already the requested value", () => {
    const block = 'lastRotated: "2026-05-12"\nperiodDays: 335';
    const out = updateLastRotated(block, "2026-05-12", "vsce-pat.md");
    expect(out).toBe(block);
  });
});
