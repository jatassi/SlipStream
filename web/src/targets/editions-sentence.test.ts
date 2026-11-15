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
    "SlipStream keeps one file at each Target, whatever its cut. A file's cut only changes its name.",
    false,
  ],
  ["Standard alone", { kind: "cuts", cuts: [standard] }, "SlipStream always gets Standard.", false],
  [
    "Standard and one other",
    { kind: "cuts", cuts: [standard, extended] },
    "SlipStream always gets Standard. It also gets Extended once a release or file of that cut turns up.",
    false,
  ],
  [
    "Standard after others",
    { kind: "cuts", cuts: [extended, standard, directors, imax] },
    "SlipStream always gets Standard. It also gets Extended, Director's Cut and IMAX once a release or file of that cut turns up.",
    false,
  ],
  [
    "one cut and no Standard",
    { kind: "cuts", cuts: [directors] },
    "SlipStream searches for Director's Cut every time, even if no release of it has turned up. It skips releases that name no cut, because they count as Standard.",
    true,
  ],
  [
    "several cuts and no Standard",
    { kind: "cuts", cuts: [extended, directors] },
    "SlipStream searches for Extended and Director's Cut every time, even if no release of them has turned up. It skips releases that name no cut, because they count as Standard.",
    true,
  ],
])("%s", (_shape, list, text, warning) => {
  expect(editionsSentence(list)).toEqual({ text, warning });
});
