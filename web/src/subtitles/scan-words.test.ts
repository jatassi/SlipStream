import { expect, test } from "vitest";

import { scansFirst, scansLeft, waitsForScan } from "./scan-words";

test("the audio scan phrases count files and never name the Speech Map", () => {
  expect(scansLeft("Node A", 312)).toBe("Node A has 312 files left to scan for speech.");
  expect(scansLeft("Node A", 1)).toBe("Node A has 1 file left to scan for speech.");
  expect(scansLeft("Node A", 1204)).toBe("Node A has 1,204 files left to scan for speech.");
  expect(scansFirst(312)).toBe("312 of them need their audio scanned first.");
  expect(scansFirst(1)).toBe("1 of them needs its audio scanned first.");
  expect(waitsForScan).toBe("Searching starts once SlipStream has scanned this file's audio.");
});
