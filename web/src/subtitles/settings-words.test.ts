import { expect, test } from "vitest";

import type { SubtitlesProvider } from "../api/gen/model";
import { gestdown, movies, subdl, subsource, tv } from "../test/subtitles";
import { providerLine } from "./settings-words";

const clock = { now: new Date("2026-03-02T12:00:00Z"), zone: "UTC" };
const anHourAgo = "2026-03-02T11:00:00Z";
const inAnHour = "2026-03-02T13:00:00Z";

const downloads = { used: 31, limit: 1250, remaining: 19, resets: inAnHour, exact: true };

const states: readonly [string, SubtitlesProvider, readonly string[], readonly string[]][] = [
  ["a keyless provider ready", gestdown, ["TV", "No key"], ["Movies", "Ready"]],
  ["a provider without its key", subdl, ["SubDL", "key", "Movies and TV"], []],
  [
    "a provider ready with downloads counted",
    {
      ...subdl,
      hasKey: true,
      allowance: { at: anHourAgo, searches: null, downloads },
      state: { kind: "ready", since: null, until: null, words: null },
    },
    ["Ready", "19", "1,250", "1:00"],
    ["SubDL", "No key"],
  ],
  [
    "a provider out of downloads",
    { ...subdl, hasKey: true, state: { kind: "out", since: null, until: inAnHour, words: null } },
    ["SubDL", "out of downloads", "1:00"],
    [],
  ],
  [
    "a provider that rejected the key",
    {
      ...subsource,
      hasKey: true,
      state: { kind: "rejected", since: anHourAgo, until: null, words: "Invalid API key" },
    },
    ["SubSource", "11:00", '"Invalid API key"', "Replace the key"],
    [],
  ],
  [
    "a keyless provider failing",
    { ...gestdown, state: { kind: "failing", since: anHourAgo, until: null, words: "timeout" } },
    ["Gestdown", "11:00", "retries", "TV", "No key"],
    ["timeout"],
  ],
  [
    "a provider turned off",
    { ...subsource, on: false, state: { kind: "off", since: null, until: null, words: null } },
    ["Off", "SubSource"],
    [],
  ],
];

test("each provider line names the facts of its state", () => {
  for (const [state, provider, names, omits] of states) {
    const line = providerLine(provider, [movies, tv], clock);
    expect(line, state).toMatch(/^[A-Z].*\.$/u);
    for (const fact of names) {
      expect(line, state).toContain(fact);
    }
    for (const fact of omits) {
      expect(line, state).not.toContain(fact);
    }
  }
});
