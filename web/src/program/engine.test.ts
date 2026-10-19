// The TypeScript engine must reproduce the Go engine exactly (plan 6, 11.4).
// Two checks: every shared fixture's hand-worked `expect`, compared with the
// same semantics as internal/program/engine_test.go (only the keys a fixture
// states), and the Go engine's whole output for every fixture and for 60
// seeded random programs (internal/program/testdata/engine/golden, written by
// `go test ./internal/program -run TestGolden -update`).

import { readFileSync, readdirSync } from "node:fs";
import { join } from "node:path";
import { fileURLToPath } from "node:url";
import { describe, expect, test } from "vitest";
import { compute, cycleWords, findCycleIn, statusWords } from "./engine";
import type { Input, Output } from "./types";

const fixtureDir = fileURLToPath(new URL("../../../internal/program/testdata/engine/", import.meta.url));
const goldenDir = join(fixtureDir, "golden");

interface Fixture extends Input {
  name: string;
  why?: string;
  expect: Record<string, unknown>;
}

const readJSON = <T,>(path: string): T => JSON.parse(readFileSync(path, "utf8")) as T;

const fixtureFiles = readdirSync(fixtureDir).filter((f) => f.endsWith(".json")).sort();

function stated(out: Output, key: string): unknown {
  switch (key) {
    case "finish":
      return out.finish.typical_index;
    case "finish_low":
      return out.finish.low_index;
    case "finish_high":
      return out.finish.high_index;
    case "finish_dates":
      return { typical: out.finish.typical, low: out.finish.low, high: out.finish.high };
    case "complete":
      return out.finish.complete;
    case "status":
      return out.status;
    case "driving":
      return out.driving_links;
    case "cycle":
      return out.cycle;
    case "conflicts":
      return out.conflicts;
  }
  throw new Error(`unknown expect key ${JSON.stringify(key)}`);
}

// checkFixture in engine_test.go: compare only the keys a fixture states;
// "why" is prose; an empty activities object means nothing is computed.
function checkFixture(f: Fixture, out: Output): void {
  for (const [key, want] of Object.entries(f.expect)) {
    if (key === "activities") {
      const acts = want as Record<string, Record<string, unknown>>;
      if (Object.keys(acts).length === 0) expect(Object.keys(out.activities), "nothing computed").toEqual([]);
      for (const [id, fields] of Object.entries(acts)) {
        const got = out.activities[id] as unknown as Record<string, unknown> | undefined;
        expect(got, `${id} missing from output`).toBeDefined();
        for (const [k, v] of Object.entries(fields)) {
          if (k === "why") continue;
          expect(got?.[k], `${id}.${k}`).toStrictEqual(v);
        }
      }
    } else if (key === "cycle_words") {
      expect(cycleWords(f, out.cycle), key).toBe(want);
    } else {
      expect(stated(out, key), key).toStrictEqual(want);
    }
  }
}

describe("shared fixtures", () => {
  test("the fixture directory is found and complete", () => {
    expect(fixtureFiles.length).toBeGreaterThanOrEqual(22);
    expect(fixtureFiles).toContain("perf-2000.json");
  });

  for (const file of fixtureFiles) {
    test(file, () => {
      const f = readJSON<Fixture>(join(fixtureDir, file));
      expect(f.name + ".json").toBe(file);
      const out = compute(f);
      checkFixture(f, out);
      // Wholesale: every field the Go engine returns.
      const golden = readJSON<{ computed: Output; cycle_words: string }>(join(goldenDir, file));
      expect(out).toStrictEqual(golden.computed);
      expect(cycleWords(f, out.cycle)).toBe(golden.cycle_words);
    });
  }
});

describe("Go golden outputs for seeded random programs", () => {
  const cases = readJSON<{ name: string; input: Input; computed: Output; cycle_words: string }[]>(
    join(goldenDir, "random.json"),
  );
  test("cover loops, unknowns and conflicts", () => {
    expect(cases.length).toBe(60);
    expect(cases.some((c) => c.computed.cycle !== null)).toBe(true);
    expect(cases.some((c) => c.computed.conflicts.length > 0)).toBe(true);
    expect(cases.some((c) => Object.values(c.computed.activities).some((a) => a.unknown))).toBe(true);
  });
  for (const c of cases) {
    test(c.name, () => {
      const out = compute(c.input);
      expect(out).toStrictEqual(c.computed);
      expect(cycleWords(c.input, out.cycle)).toBe(c.cycle_words);
      expect(findCycleIn(c.input)).toStrictEqual(c.computed.cycle);
    });
  }
});

test("compute is pure: pooled scratch never leaks between calls", () => {
  const big = readJSON<Fixture>(join(fixtureDir, "perf-2000.json"));
  const small = readJSON<Fixture>(join(fixtureDir, "mso-conflict.json"));
  const first = compute(small);
  compute(big);
  expect(compute(small)).toStrictEqual(first);
});

// TestComputeTolerantOfBadRows: input Validate refuses never throws.
test("compute tolerates rows Validate refuses", () => {
  const out = compute({
    calendar: { start: "not a date", working_week: 999 },
    activities: [
      { id: "a", duration: 2 },
      { id: "a", duration: 9 }, // duplicate id: the first wins for links
      { id: "b", parent: "ghost", duration: 3 },
      { id: "c", parent: "b", duration: 1 }, // a third tier is scheduled as top level
    ],
    links: [
      { id: "x", pred: "a", succ: "zz" },
      { id: "y", pred: "a", succ: "b", type: "QQ" },
    ],
  });
  expect(out.finish.typical_index).toBe(9);
  expect(out.finish.typical).toBeNull();
  expect(out.status.words.startsWith("All")).toBe(true);
});

test("status words", () => {
  expect(statusWords(31, 44)).toBe("Indicative: 31 of 44 durations known");
  expect(statusWords(0, 12)).toBe("Indicative: 0 of 12 durations known");
  expect(statusWords(3, 3)).toBe("All 3 durations known");
  expect(statusWords(0, 0)).toBe("No activities yet");
});

// Plan 6.7: TypeScript under 5 ms for 2,000 activities and 4,000 links on a
// mid-range laptop. The gate is generous so a busy CI runner does not fail
// the build; the real median is printed.
test("compute on 2,000 activities stays inside its budget", () => {
  const f = readJSON<Fixture>(join(fixtureDir, "perf-2000.json"));
  expect(f.activities?.length).toBe(2000);
  expect(f.links?.length).toBe(4000);
  for (let k = 0; k < 30; k++) compute(f); // warm the JIT and the pool
  const times: number[] = [];
  for (let k = 0; k < 101; k++) {
    const t0 = performance.now();
    compute(f);
    times.push(performance.now() - t0);
  }
  times.sort((a, b) => a - b);
  const median = times[50];
  const p90 = times[90];
  console.log(`compute 2000: median ${median.toFixed(2)} ms, p90 ${p90.toFixed(2)} ms (budget 5 ms, gate 25 ms)`);
  expect(median).toBeLessThan(25);
});
