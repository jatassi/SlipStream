import { expect, test } from "vitest";

import type { FormatsPasted, FormatsPastedFormat } from "../api/gen/model";
import { outcomeWords, pasteLabel, replacedAtFirst } from "./paste-words";

const nothing = {
  works: 0,
  ownList: 0,
  unwanted: { files: 0, bytes: 0 },
  renamed: 0,
  renames: [],
  newlyWanted: 0,
  downloading: 0,
};

const added: FormatsPastedFormat = { name: "DV", outcome: "add", savedAs: "DV" };
const sameApp: FormatsPastedFormat = {
  name: "HDR",
  outcome: "keep_both",
  savedAs: "HDR (2)",
  sameName: { id: "radarr-hdr", name: "hdr", dialect: "radarr" },
};
const otherApp: FormatsPastedFormat = {
  name: "x265",
  outcome: "keep_both",
  savedAs: "x265 (2)",
  sameName: { id: "sonarr-x265", name: "x265", dialect: "sonarr" },
};

test("a paste replaces at first only the formats of a pasted name that hold the same app's values", () => {
  const pasted: FormatsPasted = {
    app: "radarr",
    formats: [added, sameApp, otherApp],
    impact: nothing,
  };
  expect(replacedAtFirst(pasted)).toEqual(["radarr-hdr"]);
});

test.each<[string, FormatsPastedFormat, string]>([
  ["a new name", added, "Adds it."],
  [
    "a replaced format",
    { ...sameApp, outcome: "replace", savedAs: "HDR" },
    "Replaces the hdr you have.",
  ],
  ["a format kept beside one of its name", sameApp, "Adds it as HDR (2), beside the hdr you have."],
])("%s", (_, format, want) => {
  expect(outcomeWords(format)).toBe(want);
});

test.each<[FormatsPastedFormat[], string]>([
  [[added], "Add 1 Custom Format"],
  [[added, sameApp], "Add 2 Custom Formats"],
  [[{ ...sameApp, outcome: "replace" }], "Replace 1 Custom Format"],
  [[added, otherApp, { ...sameApp, outcome: "replace" }], "Replace 1 and add 2"],
])("the save button for %j", (formats, want) => {
  expect(pasteLabel(formats)).toBe(want);
});
