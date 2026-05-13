#!/usr/bin/env bun
/**
 * Open a GitHub issue when a long-lived secret is due for rotation.
 *
 * The source of truth is the `rotations:` table in the front matter
 * of `docs/development/secret-rotations.md`. For each entry the
 * script computes (today - last-rotated). If that exceeds
 * `period-days` minus the reminder window (30 days), the script
 * opens a single labelled issue per entry, with a stable title so
 * reruns are idempotent.
 *
 * The script does not close issues directly. The reminder issue
 * body points the maintainer at the `Record Secret Rotation`
 * workflow, which opens a PR that updates `last-rotated` and
 * includes `Closes #N`. Merging that PR records the rotation date
 * AND closes the reminder in one step. The next scheduled run
 * sees the new date and stays quiet until the next expiry window.
 *
 * Exit codes:
 * - 0: ran cleanly (regardless of whether any issue was opened)
 * - 1: doc missing / malformed / `gh` failure / unknown entry kind
 */

import { $ } from "bun";
import { resolve } from "node:path";
import { parse as yamlParse } from "yaml";

const REPO_ROOT = resolve(import.meta.dir, "..", "..");
const ROTATION_DOC = resolve(REPO_ROOT, "docs/development/secret-rotations.md");
const REMINDER_WINDOW_DAYS = 30;
const ISSUE_LABEL = "secret-rotation";
const ASSIGNEE = "jeduden";

interface RotationEntry {
  name: string;
  "last-rotated": string;
  "period-days": number;
  provider: string;
  "issuer-url": string;
  "used-by": string;
  scope: string;
}

interface FrontMatter {
  rotations?: unknown;
}

/** Compute today's date in UTC. The cron schedule is UTC, so the
 * computed due-state must match the workflow's wall clock. */
function utcToday(): Date {
  const now = new Date();
  return new Date(Date.UTC(
    now.getUTCFullYear(),
    now.getUTCMonth(),
    now.getUTCDate(),
  ));
}

/** Days between two UTC midnights. Truncates to integer. */
function daysBetween(later: Date, earlier: Date): number {
  const msPerDay = 24 * 60 * 60 * 1000;
  return Math.floor((later.getTime() - earlier.getTime()) / msPerDay);
}

/** Extract the YAML front matter block from a markdown file. */
function frontMatter(text: string): FrontMatter {
  if (!text.startsWith("---\n")) {
    throw new SystemExit(`${ROTATION_DOC}: no front matter (must start with '---\\n')`);
  }
  const end = text.indexOf("\n---\n", 4);
  if (end === -1) {
    throw new SystemExit(`${ROTATION_DOC}: unterminated front matter`);
  }
  const parsed = yamlParse(text.slice(4, end));
  if (parsed === null || typeof parsed !== "object" || Array.isArray(parsed)) {
    throw new SystemExit(`${ROTATION_DOC}: front matter is not a mapping`);
  }
  return parsed as FrontMatter;
}

class SystemExit extends Error {}

type DueState = "ok" | "due" | "overdue";

function dueState(today: Date, lastRotated: Date, periodDays: number): { status: DueState; days: number } {
  const dueOn = new Date(lastRotated);
  dueOn.setUTCDate(dueOn.getUTCDate() + periodDays);
  const days = daysBetween(dueOn, today);
  if (days < 0) return { status: "overdue", days };
  if (days <= REMINDER_WINDOW_DAYS) return { status: "due", days };
  return { status: "ok", days };
}

/** Return the absolute GitHub URL of this repository. */
function repoUrl(): string {
  const server = (process.env.GITHUB_SERVER_URL ?? "https://github.com").replace(/\/+$/, "");
  const repo = process.env.GITHUB_REPOSITORY ?? "jeduden/mdsmith";
  return `${server}/${repo}`;
}

interface OpenIssue { number: number; title: string }

/** Return the number of an open issue whose title exactly matches. */
async function existingOpenIssue(title: string): Promise<number | null> {
  const out = await $`gh issue list \
    --state open \
    --label ${ISSUE_LABEL} \
    --json number,title \
    --limit 100`.text();
  const issues = JSON.parse(out || "[]") as OpenIssue[];
  for (const issue of issues) {
    if (issue.title === title) return issue.number;
  }
  return null;
}

/** Idempotently create the `secret-rotation` label. */
let labelEnsured = false;
async function ensureLabel(): Promise<void> {
  if (labelEnsured) return;
  await $`gh label create ${ISSUE_LABEL} \
    --force \
    --color C5DEF5 \
    --description ${"Long-lived secret is due (or overdue) for rotation"}`.quiet();
  labelEnsured = true;
}

function issueBody(entry: RotationEntry, status: DueState, days: number): string {
  const headline = status === "overdue"
    ? `\`${entry.name}\` is OVERDUE by ${-days} days.`
    : `\`${entry.name}\` is due in ${days} days.`;
  const docUrl = `${repoUrl()}/blob/main/docs/development/secret-rotations.md`;
  const reminderUrl = `${repoUrl()}/blob/main/.github/workflows/secret-rotation-reminder.yml`;
  const recordUrl = `${repoUrl()}/actions/workflows/record-secret-rotation.yml`;
  return [
    `${headline}`,
    ``,
    `| Field         | Value                                            |`,
    `|---------------|--------------------------------------------------|`,
    `| Provider      | ${entry.provider}                                       |`,
    `| Issuer URL    | <${entry["issuer-url"]}>                                   |`,
    `| Used by       | ${entry["used-by"]}                                        |`,
    `| Scope         | ${entry.scope}                                          |`,
    `| Last rotated  | ${entry["last-rotated"]}                                           |`,
    `| Period (days) | ${entry["period-days"]}                                         |`,
    ``,
    `Rotation procedure for the \`${entry.name}\` section:`,
    `${docUrl}`,
    ``,
    `After rotating the credential at the issuer, do not`,
    `hand-edit the front matter or close this issue.`,
    `Instead, run the **Record Secret Rotation** workflow:`,
    `${recordUrl}`,
    ``,
    `Pick \`${entry.name}\` from the dropdown and click \`Run workflow\`.`,
    `The workflow opens a PR that updates \`last-rotated\``,
    `and includes \`Closes #\` referencing this issue, so`,
    `the merge both records the rotation and closes this`,
    `reminder in one step.`,
    ``,
    `This reminder was opened automatically by ${reminderUrl}.`,
    "",
  ].join("\n");
}

function isMapping(value: unknown): value is Record<string, unknown> {
  return typeof value === "object" && value !== null && !Array.isArray(value);
}

async function main(): Promise<number> {
  const text = await Bun.file(ROTATION_DOC).text();
  const fm = frontMatter(text);
  const rotations = fm.rotations;
  if (!Array.isArray(rotations) || rotations.length === 0) {
    throw new SystemExit(`${ROTATION_DOC}: front matter has no \`rotations:\` list`);
  }

  const today = utcToday();
  const opened: string[] = [];
  const skipped: string[] = [];
  const required: (keyof RotationEntry)[] = [
    "name", "last-rotated", "period-days", "provider", "issuer-url", "used-by", "scope",
  ];

  for (const raw of rotations) {
    if (!isMapping(raw)) {
      throw new SystemExit(`rotation entry is not a mapping: ${JSON.stringify(raw)}`);
    }
    for (const key of required) {
      if (!(key in raw)) {
        throw new SystemExit(
          `rotation entry missing \`${key}\`: ${JSON.stringify(raw)}`,
        );
      }
    }
    const entry = raw as unknown as RotationEntry;
    const name = entry.name;
    const lastStr = String(entry["last-rotated"]);
    if (!/^\d{4}-\d{2}-\d{2}$/.test(lastStr)) {
      throw new SystemExit(
        `rotation entry '${name}': \`last-rotated\` is not a valid ISO-8601 date (${JSON.stringify(lastStr)})`,
      );
    }
    const lastRotated = new Date(`${lastStr}T00:00:00Z`);
    if (Number.isNaN(lastRotated.getTime())) {
      throw new SystemExit(
        `rotation entry '${name}': \`last-rotated\` is not a valid ISO-8601 date (${JSON.stringify(lastStr)})`,
      );
    }
    const periodDays = Number(entry["period-days"]);
    if (!Number.isInteger(periodDays)) {
      throw new SystemExit(
        `rotation entry '${name}': \`period-days\` is not an integer (${JSON.stringify(entry["period-days"])})`,
      );
    }
    const { status, days } = dueState(today, lastRotated, periodDays);
    if (status === "ok") continue;
    const title = `Rotate ${name} (last rotated ${lastStr})`;
    if (await existingOpenIssue(title) !== null) {
      skipped.push(name);
      continue;
    }
    await ensureLabel();
    const body = issueBody(entry, status, days);
    await $`gh issue create \
      --title ${title} \
      --body ${body} \
      --label ${ISSUE_LABEL} \
      --assignee ${ASSIGNEE}`.quiet();
    opened.push(name);
  }

  if (opened.length) {
    console.log(`opened secret-rotation reminders for: ${opened.join(", ")}`);
  }
  if (skipped.length) {
    console.log(`existing open reminders (skipped): ${skipped.join(", ")}`);
  }
  if (!opened.length && !skipped.length) {
    console.log("no secrets due for rotation");
  }
  return 0;
}

try {
  process.exit(await main());
} catch (err) {
  if (err instanceof SystemExit) {
    process.stderr.write(`${err.message}\n`);
    process.exit(1);
  }
  throw err;
}
