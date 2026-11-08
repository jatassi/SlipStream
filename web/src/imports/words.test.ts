import { expect, test } from "vitest";

import type { ImportsReason, ImportsReasonCode, WorksContentType } from "../api/gen/model";
import { cinema } from "../test/library";
import { brazil, filmsVault, severanceEntry, shows } from "../test/pending-imports";
import { reasonOf, reasonsOf } from "./words";

const every: {
  readonly [C in ImportsReasonCode]: readonly [ImportsReason, WorksContentType, string];
} = {
  several: [
    { code: "several", title: "Halloween", count: 3 },
    cinema,
    "3 films match Halloween. The name has no year.",
  ],
  no_match: [
    { code: "no_match", title: "Brazil", year: 1985 },
    cinema,
    "No film matches Brazil (1985).",
  ],
  no_year: [
    { code: "no_year", title: "Halloween" },
    cinema,
    "One film matches Halloween, but the name has no year.",
  ],
  elsewhere: [
    { code: "elsewhere", title: "Heat", year: 1995, root: filmsVault },
    cinema,
    "Heat (1995) is in the Films library root on vault. A Work's files all live in one root. Move these files there, or ignore them.",
  ],
  unlisted: [
    { code: "unlisted", title: "Heat", year: 1995 },
    cinema,
    "TMDB no longer lists Heat (1995).",
  ],
  no_number: [{ code: "no_number" }, shows, "The file name has no episode number."],
  no_such_item: [
    { code: "no_such_item", names: "S03E14" },
    shows,
    "This show has no episode S03E14.",
  ],
  several_items: [{ code: "several_items" }, shows, "The file holds several episodes."],
  unreadable: [
    { code: "unreadable", line: "Invalid data found when processing input" },
    cinema,
    "ffprobe can't read this file. Replace it, then scan again.",
  ],
  left: [
    { code: "left" },
    cinema,
    "This film was deleted from the library and its files were left on disk.",
  ],
  no_rule: [{ code: "no_rule" }, shows, "SlipStream can't tell which episode this file holds."],
};

test("every reason a file waits reads as a sentence in the type's words", () => {
  for (const [reason, type, sentence] of Object.values(every)) {
    expect(reasonOf(reason, type)).toBe(sentence);
  }
});

test("an entry's reasons read once each, counting the files that share one", () => {
  expect(reasonsOf(brazil, cinema)).toEqual([
    "No film matches Brazil (1985).",
    "ffprobe can't read this file. Replace it, then scan again.",
  ]);
  const twice = {
    ...severanceEntry,
    files: severanceEntry.files.map((file) =>
      Object.assign({}, file, { reason: { code: "no_number" as const } }),
    ),
  };
  expect(reasonsOf(twice, shows)).toEqual(["2 file names have no episode number."]);
});
