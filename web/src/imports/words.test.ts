import { expect, test } from "vitest";

import type { ImportsReason, ImportsReasonCode, WorksContentType } from "../api/gen/model";
import { cinema } from "../test/library";
import { brazil, filmsVault, severanceEntry, shows } from "../test/pending-imports";
import { reasonOf, reasonsOf } from "./words";

const worded = (code: ImportsReasonCode): readonly [ImportsReason, WorksContentType, string] => [
  { code, sentence: `The server words ${code}.` },
  cinema,
  `The server words ${code}.`,
];

const every: {
  readonly [C in ImportsReasonCode]: readonly [ImportsReason, WorksContentType, string];
} = {
  several: [
    { code: "several", sentence: null, title: "Halloween", count: 3 },
    cinema,
    "3 films match Halloween. The name has no year.",
  ],
  no_match: [
    { code: "no_match", sentence: null, title: "Brazil", year: 1985 },
    cinema,
    "No film matches Brazil (1985).",
  ],
  no_year: [
    { code: "no_year", sentence: null, title: "Halloween" },
    cinema,
    "One film matches Halloween, but the name has no year.",
  ],
  elsewhere: [
    { code: "elsewhere", sentence: null, title: "Heat", year: 1995, root: filmsVault },
    cinema,
    "Heat (1995) is in the Films library root on vault. A Work's files all live in one root. Move these files there, or ignore them.",
  ],
  unlisted: [
    { code: "unlisted", sentence: null, title: "Heat", year: 1995 },
    cinema,
    "TMDB no longer lists Heat (1995).",
  ],
  no_number: [{ code: "no_number", sentence: null }, shows, "The file name has no episode number."],
  no_such_item: [
    { code: "no_such_item", sentence: null, names: "S03E14" },
    shows,
    "This show has no episode S03E14.",
  ],
  unreadable: [
    { code: "unreadable", sentence: null, line: "Invalid data found when processing input" },
    cinema,
    "ffprobe can't read this file. Replace it, then scan again.",
  ],
  left: [
    { code: "left", sentence: null },
    cinema,
    "This film was deleted from the library and its files were left on disk.",
  ],
  no_rule: [
    { code: "no_rule", sentence: null },
    shows,
    "SlipStream can't tell which episode this file holds.",
  ],
  sample: worded("sample"),
  bonus: worded("bonus"),
  hidden_folder: worded("hidden_folder"),
  too_small: worded("too_small"),
  extension: worded("extension"),
  runtime_sample: worded("runtime_sample"),
  sibling_better: worded("sibling_better"),
  kept_better: worded("kept_better"),
  meets_nothing: worded("meets_nothing"),
  not_an_upgrade: worded("not_an_upgrade"),
  edition_not_wanted: worded("edition_not_wanted"),
  other_items: worded("other_items"),
  parts: worded("parts"),
  name_taken: worded("name_taken"),
  name_too_long: worded("name_too_long"),
  settings: worded("settings"),
  busy: worded("busy"),
  confirm: worded("confirm"),
  import_failed: worded("import_failed"),
  root_changed: worded("root_changed"),
  by_admin: worded("by_admin"),
  removed: worded("removed"),
  canceled: worded("canceled"),
  blocklisted: worded("blocklisted"),
  work_deleted: worded("work_deleted"),
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
      Object.assign({}, file, { reason: { code: "no_number" as const, sentence: null } }),
    ),
  };
  expect(reasonsOf(twice, shows)).toEqual(["2 file names have no episode number."]);
});
