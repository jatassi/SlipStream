import { expect, test } from "vitest";

import { subtitleLines } from "./impact-words";

const english = { language: "en", forced: false };

test("a gained language counts its searches, and says which wait on an audio scan", () => {
  expect(
    subtitleLines({ gained: [{ ...english, searches: 1, needScan: 1 }], dropped: [], days: 0 }),
  ).toEqual(["Saving searches English subtitles for 1 file. SlipStream scans its audio first."]);
  expect(
    subtitleLines({ gained: [{ ...english, searches: 3, needScan: 3 }], dropped: [], days: 0 }),
  ).toEqual(["Saving searches English subtitles for 3 files. SlipStream scans their audio first."]);
  expect(
    subtitleLines({ gained: [{ ...english, searches: 3, needScan: 1 }], dropped: [], days: 2 }),
  ).toEqual([
    "Saving searches English subtitles for 3 files. 1 of them needs its audio scanned first.",
    "At today's downloads, the last of these arrives in about 2 days.",
  ]);
});
