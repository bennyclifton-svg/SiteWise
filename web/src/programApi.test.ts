import { expect, test } from "vitest";
import { needsRefetch, newerAnnouncement } from "./programApi";

test("a newer version of the shown program refetches once", () => {
  const remote = { programId: "p1", version: 5 };
  expect(needsRefetch(remote, { id: "p1", version: 4 }, null)).toBe(true);
  expect(needsRefetch(remote, { id: "p1", version: 5 }, null)).toBe(false);
  // Fetched for, the GET answered a lower version: no second fetch, no loop.
  expect(needsRefetch(remote, { id: "p1", version: 3 }, remote)).toBe(false);
  expect(needsRefetch({ programId: "", version: 0 }, { id: "p1", version: 1 }, null)).toBe(false);
});

test("another program id refetches even at a lower version, once", () => {
  const remote = { programId: "p2", version: 1 };
  expect(needsRefetch(remote, { id: "p1", version: 9 }, null)).toBe(true);
  expect(needsRefetch(remote, { id: "p1", version: 9 }, remote)).toBe(false);
});

test("announcements keep the newest version; a new program id restarts them", () => {
  let cur = newerAnnouncement({ programId: "", version: 0 }, "p1", 4);
  expect(cur).toEqual({ programId: "p1", version: 4 });
  expect(newerAnnouncement(cur, "p1", 3)).toBe(cur);
  cur = newerAnnouncement(cur, "p2", 1);
  expect(cur).toEqual({ programId: "p2", version: 1 });
});
