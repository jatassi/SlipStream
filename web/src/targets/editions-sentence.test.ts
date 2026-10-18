import { expect, test } from "vitest";

import { type Cut, type EditionList, editionsSentence } from "./editions-sentence";

const standard: Cut = { key: "standard", name: "Standard" };
const extended: Cut = { key: "extended", name: "Extended" };
const directors: Cut = { key: "directors-cut", name: "Director's Cut" };
const imax: Cut = { key: "imax", name: "IMAX" };

test.each<[string, EditionList, string, boolean]>([
  [
    "any cut",
    { kind: "any" },
    "Each Item wants one file at every Target, of whatever cut. A file's cut only sets its name.",
    false,
  ],
  [
    "Standard alone",
    { kind: "cuts", cuts: [standard] },
    "Every Item always wants Standard.",
    false,
  ],
  [
    "Standard and one other",
    { kind: "cuts", cuts: [standard, extended] },
    "Every Item always wants Standard. It wants Extended only once a release or file of that cut has been seen for it.",
    false,
  ],
  [
    "Standard after others",
    { kind: "cuts", cuts: [extended, standard, directors, imax] },
    "Every Item always wants Standard. It wants Extended, Director's Cut and IMAX only once a release or file of that cut has been seen for it.",
    false,
  ],
  [
    "one cut and no Standard",
    { kind: "cuts", cuts: [directors] },
    "Every Item wants Director's Cut outright, even where no release of it has been seen. A release that names no cut is Standard, which this list doesn't want.",
    true,
  ],
  [
    "several cuts and no Standard",
    { kind: "cuts", cuts: [extended, directors] },
    "Every Item wants Extended and Director's Cut outright, even where no release of them has been seen. A release that names no cut is Standard, which this list doesn't want.",
    true,
  ],
])("%s", (_shape, list, text, warning) => {
  expect(editionsSentence(list)).toEqual({ text, warning });
});
