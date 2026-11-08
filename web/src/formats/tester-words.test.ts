import { expect, test } from "vitest";

import { ApiError } from "../api/errors";
import type { FormatsExplained, FormatsParsed, FormatsReadFacts } from "../api/gen/model";
import {
  conditionNotes,
  conditionResult,
  draftProblems,
  factRows,
  formatGroups,
  namesIn,
  namesProblem,
  verdictWords,
} from "./tester-words";

const plain: FormatsReadFacts = {
  quality: null,
  source: null,
  resolution: null,
  modifier: null,
  revision: null,
  group: null,
  cut: "standard",
  edition: null,
  hdr: [],
  video: null,
  audio: [],
  channels: [],
  languages: [],
  ids: {},
  titles: [],
  year: null,
  marks: [],
  span: "",
};

test("pasted text gives one name per non-blank line, trimmed, from any platform's line ends", () => {
  expect(
    namesIn("  Heat.1995.1080p.BluRay-GRP \r\n\n\tC:\\Films\\Heat (1995)\\heat.mkv\n   \n"),
  ).toEqual(["Heat.1995.1080p.BluRay-GRP", "C:\\Films\\Heat (1995)\\heat.mkv"]);
});

test("more names than one test takes are refused in words of the kind pasted", () => {
  const many = Array.from({ length: 101 }, (_, index) => `Name.${String(index)}`);
  expect(namesProblem(many.slice(0, 100), "release")).toBeUndefined();
  expect(namesProblem(many, "release")).toBe("This has 101 names. Test 100 or fewer at a time.");
  expect(namesProblem(many, "file")).toBe("This has 101 paths. Test 100 or fewer at a time.");
});

test("the facts list only what the name gives, in reading order", () => {
  expect(
    factRows({
      ...plain,
      quality: "WEBDL-1080p",
      source: "WEB-DL",
      resolution: "1080p",
      revision: "REAL v2 REPACK",
      group: "FLUX",
      cut: "extended",
      hdr: ["Dolby Vision", "HDR10"],
      audio: ["DD+"],
      channels: ["5.1"],
      ids: { tmdbid: "949" },
      titles: ["Heat", "Heat AKA"],
      year: 1995,
      marks: ["S01E02"],
    }),
  ).toEqual([
    { label: "Titles", value: "Heat, Heat AKA" },
    { label: "Year", value: "1995" },
    { label: "Numbering", value: "S01E02" },
    { label: "Quality", value: "WEBDL-1080p" },
    { label: "Source", value: "WEB-DL" },
    { label: "Resolution", value: "1080p" },
    { label: "Revision", value: "REAL v2 REPACK" },
    { label: "Release group", value: "FLUX" },
    { label: "Cut", value: "extended" },
    { label: "HDR", value: "Dolby Vision, HDR10" },
    { label: "Audio", value: "DD+" },
    { label: "Channels", value: "5.1" },
    { label: "Id tags", value: "tmdbid 949" },
  ]);
  expect(factRows(plain)).toEqual([]);
});

const format = (name: string, matched: boolean, draft = false): FormatsExplained => ({
  id: name,
  name,
  matched,
  draft,
  timedOut: false,
  conditions: [],
});

test("the matched formats and the draft show their conditions, and the rest are only named", () => {
  const { shown, others } = formatGroups([
    format("x265", false),
    format("DV", true),
    format("Draft", false, true),
    format("HDR10+", false),
  ]);
  expect(shown.map((each) => each.name)).toEqual(["DV", "Draft"]);
  expect(others).toEqual(["x265", "HDR10+"]);
});

test("a condition's result reads after Negate, and a timed-out pattern fails", () => {
  const condition = { name: "DV", kind: "ReleaseTitleSpecification", negate: true, required: true };
  expect(conditionResult({ ...condition, matched: true, timedOut: false })).toBe("Passes");
  expect(conditionResult({ ...condition, matched: false, timedOut: false })).toBe("Fails");
  expect(conditionResult({ ...condition, matched: false, timedOut: true })).toBe(
    "Fails, because its pattern ran past its time limit",
  );
  expect(conditionNotes({ ...condition, matched: true, timedOut: false })).toEqual([
    "Negate",
    "Required",
  ]);
});

const heat = { id: "heat", label: "Heat" };

const result = (works: number, judgedAgainst = works === 1): FormatsParsed => ({
  name: "Heat.1995.1080p.BluRay-GRP",
  sentence: "",
  words: "",
  facts: plain,
  work: judgedAgainst ? heat : null,
  works: Array.from({ length: works }, () => heat),
  covers: [],
  formats: [],
  targets: [],
});

test("a name judged against one Work fills, and one matching several Works or none is accepted", () => {
  expect(verdictWords(result(1), true)).toBe("Would fill");
  expect(verdictWords(result(1), false)).toBe("Wouldn't fill");
  expect(verdictWords(result(0), true)).toBe("Accepts it");
  expect(verdictWords(result(2), false)).toBe("Doesn't accept it");
  expect(verdictWords(result(0, true), false)).toBe("Wouldn't fill");
});

test("a draft's compile refusal lands under the editor's own field", () => {
  const refused = new ApiError({
    status: 400,
    code: "format_pattern_not_compiled",
    title: "The pattern in this condition doesn't compile",
    errors: [
      {
        location: "body.draft.conditions[1].fields.value",
        message: 'The pattern in "DV" doesn\'t compile.',
      },
    ],
  });
  expect(draftProblems(refused)).toEqual({
    "conditions[1].fields.value": 'The pattern in "DV" doesn\'t compile.',
  });
});
