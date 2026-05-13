import { afterEach, describe, expect, test } from "bun:test";
import {
  daysBetween,
  dueState,
  isIsoDate,
  issueBody,
  parseFrontMatter,
  repoUrl,
  SystemExit,
  utcToday,
  validateRotationEntry,
} from "./check-secret-rotations";

describe("isIsoDate", () => {
  test("accepts a real calendar date", () => {
    expect(isIsoDate("2026-05-13")).toBe(true);
  });
  test("rejects a calendar-invalid date that Date.parse normalizes", () => {
    expect(isIsoDate("2026-02-31")).toBe(false);
  });
  test("rejects malformed input", () => {
    expect(isIsoDate("2026-5-13")).toBe(false);
    expect(isIsoDate("not-a-date")).toBe(false);
    expect(isIsoDate("")).toBe(false);
  });
  test("rejects out-of-range months and days", () => {
    expect(isIsoDate("2026-13-01")).toBe(false);
    expect(isIsoDate("2026-00-15")).toBe(false);
    expect(isIsoDate("2026-05-32")).toBe(false);
  });
});

describe("utcToday", () => {
  test("returns a UTC midnight", () => {
    const today = utcToday();
    expect(today.getUTCHours()).toBe(0);
    expect(today.getUTCMinutes()).toBe(0);
    expect(today.getUTCSeconds()).toBe(0);
    expect(today.getUTCMilliseconds()).toBe(0);
  });
});

describe("daysBetween", () => {
  const day = 24 * 60 * 60 * 1000;
  test("zero on the same day", () => {
    const a = new Date(Date.UTC(2026, 0, 1));
    expect(daysBetween(a, a)).toBe(0);
  });
  test("positive when later > earlier", () => {
    const earlier = new Date(Date.UTC(2026, 0, 1));
    const later = new Date(earlier.getTime() + 7 * day);
    expect(daysBetween(later, earlier)).toBe(7);
  });
  test("negative when later < earlier", () => {
    const earlier = new Date(Date.UTC(2026, 0, 10));
    const later = new Date(earlier.getTime() - 3 * day);
    expect(daysBetween(later, earlier)).toBe(-3);
  });
});

describe("dueState", () => {
  const lastRotated = new Date(Date.UTC(2026, 0, 1));
  const periodDays = 30;
  // dueOn = lastRotated + 30 days = 2026-01-31

  test("ok when today is well before due window", () => {
    const today = new Date(Date.UTC(2025, 11, 1));
    const r = dueState(today, lastRotated, periodDays);
    expect(r.status).toBe("ok");
  });
  test("due when within REMINDER_WINDOW_DAYS (30) of expiry", () => {
    // 15 days before dueOn → due
    const today = new Date(Date.UTC(2026, 0, 16));
    const r = dueState(today, lastRotated, periodDays);
    expect(r.status).toBe("due");
    expect(r.daysUntilDue).toBe(15);
  });
  test("due on the due date with daysUntilDue=0", () => {
    const today = new Date(Date.UTC(2026, 0, 31));
    const r = dueState(today, lastRotated, periodDays);
    expect(r.status).toBe("due");
    expect(r.daysUntilDue).toBe(0);
  });
  test("overdue once past due with negative daysUntilDue", () => {
    const today = new Date(Date.UTC(2026, 1, 5));
    const r = dueState(today, lastRotated, periodDays);
    expect(r.status).toBe("overdue");
    expect(r.daysUntilDue).toBe(-5);
  });
});

describe("parseFrontMatter", () => {
  test("parses a well-formed front-matter block", () => {
    const text = '---\ntitle: VSCE_PAT\nperiodDays: 335\n---\n# body\n';
    const fm = parseFrontMatter(text, "test.md");
    expect(fm.title).toBe("VSCE_PAT");
    expect(fm.periodDays).toBe(335);
  });
  test("rejects a file without front matter", () => {
    expect(() => parseFrontMatter("no front matter here\n", "test.md"))
      .toThrow(SystemExit);
  });
  test("rejects an unterminated front-matter block", () => {
    expect(() => parseFrontMatter("---\ntitle: X\nbody but no close\n", "test.md"))
      .toThrow(SystemExit);
  });
  test("rejects a front-matter block whose root is not a mapping", () => {
    expect(() => parseFrontMatter("---\n- list\n- of\n- scalars\n---\n", "test.md"))
      .toThrow(SystemExit);
  });
});

describe("repoUrl", () => {
  const savedServer = process.env.GITHUB_SERVER_URL;
  const savedRepo = process.env.GITHUB_REPOSITORY;
  afterEach(() => {
    if (savedServer === undefined) delete process.env.GITHUB_SERVER_URL;
    else process.env.GITHUB_SERVER_URL = savedServer;
    if (savedRepo === undefined) delete process.env.GITHUB_REPOSITORY;
    else process.env.GITHUB_REPOSITORY = savedRepo;
  });

  test("uses GITHUB_SERVER_URL and GITHUB_REPOSITORY when set", () => {
    process.env.GITHUB_SERVER_URL = "https://github.example.com";
    process.env.GITHUB_REPOSITORY = "acme/widget";
    expect(repoUrl()).toBe("https://github.example.com/acme/widget");
  });
  test("strips trailing slashes from server URL", () => {
    process.env.GITHUB_SERVER_URL = "https://github.example.com///";
    process.env.GITHUB_REPOSITORY = "acme/widget";
    expect(repoUrl()).toBe("https://github.example.com/acme/widget");
  });
  test("falls back to github.com and jeduden/mdsmith when env is empty", () => {
    delete process.env.GITHUB_SERVER_URL;
    delete process.env.GITHUB_REPOSITORY;
    expect(repoUrl()).toBe("https://github.com/jeduden/mdsmith");
  });
});

describe("issueBody", () => {
  const entry = {
    title: "VSCE_PAT",
    lastRotated: "2026-05-12",
    periodDays: 335,
    provider: "Azure DevOps",
    issuerUrl: "https://dev.azure.com",
    usedBy: "release.yml",
    scope: "Marketplace > Manage",
  };

  test("renders an overdue headline with a positive day count", () => {
    const body = issueBody(entry, "vsce-pat.md", { status: "overdue", daysUntilDue: -7 });
    expect(body).toContain("OVERDUE by 7 days");
  });
  test("renders a 'due today' headline when daysUntilDue is 0", () => {
    const body = issueBody(entry, "vsce-pat.md", { status: "due", daysUntilDue: 0 });
    expect(body).toContain("is due today");
  });
  test("renders a 'due in N days' headline when in the reminder window", () => {
    const body = issueBody(entry, "vsce-pat.md", { status: "due", daysUntilDue: 15 });
    expect(body).toContain("is due in 15 days");
  });
  test("includes the field table and rotation procedure link", () => {
    const body = issueBody(entry, "vsce-pat.md", { status: "due", daysUntilDue: 5 });
    expect(body).toContain("| Provider | Azure DevOps |");
    expect(body).toContain("| Period (days) | 335 |");
    expect(body).toContain("vsce-pat.md");
  });
});

describe("validateRotationEntry", () => {
  const good = {
    title: "VSCE_PAT",
    lastRotated: "2026-05-12",
    periodDays: 335,
    provider: "Azure DevOps",
    issuerUrl: "https://dev.azure.com",
    usedBy: "release.yml",
    scope: "Marketplace > Manage",
  };

  test("accepts a complete, well-formed entry", () => {
    const out = validateRotationEntry(good, "vsce-pat.md");
    expect(out.title).toBe("VSCE_PAT");
    expect(out.periodDays).toBe(335);
  });

  test("rejects a missing required key with a path-prefixed message", () => {
    const { title, ...rest } = good;
    expect(() => validateRotationEntry(rest, "x.md")).toThrow(/x\.md.*title/);
  });

  test("rejects a calendar-invalid lastRotated", () => {
    expect(() => validateRotationEntry({ ...good, lastRotated: "2026-02-31" }, "x.md"))
      .toThrow(/lastRotated.*ISO-8601/);
  });

  test("rejects a non-integer periodDays", () => {
    expect(() => validateRotationEntry({ ...good, periodDays: "soon" }, "x.md"))
      .toThrow(/periodDays.*integer/);
  });

  test("rejects a zero or negative periodDays", () => {
    expect(() => validateRotationEntry({ ...good, periodDays: 0 }, "x.md"))
      .toThrow(/periodDays.*positive/);
    expect(() => validateRotationEntry({ ...good, periodDays: -1 }, "x.md"))
      .toThrow(/periodDays.*positive/);
  });
});
