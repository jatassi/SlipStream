import { expect, test } from "vitest";

import { pathDiff } from "./path-diff";

const marked = (parts: { text: string; changed: boolean }[]) =>
  parts.map(({ text, changed }) => (changed ? `[${text}]` : text)).join("");

test("a new title marks the folder and the name, and keeps what they share", () => {
  const { from, to } = pathDiff(
    "Films/Heat 1995/Heat 1995 - 4K.mkv",
    "Films/Heat (1995)/Heat (1995) - 4K.mkv",
  );
  expect(marked(from)).toBe("Films/[Heat 1995/Heat 1995 - 4K.mkv]");
  expect(marked(to)).toBe("Films/[Heat (1995)/Heat (1995) - 4K.mkv]");
});

test("a file moving out of its level folder marks only that folder", () => {
  const { from, to } = pathDiff("Shelf/Volume 01/Shelf - 1x01.mkv", "Shelf/Shelf - 1x01.mkv");
  expect(marked(from)).toBe("Shelf/[Volume 01]/Shelf - 1x01.mkv");
  expect(marked(to)).toBe("Shelf/Shelf - 1x01.mkv");
});

test("a new name in the same folder marks only the name", () => {
  const { from, to } = pathDiff(
    "Heat (1995)/Heat (1995) - 4K.mkv",
    "Heat (1995)/Heat (1995) - 4K HDR.mkv",
  );
  expect(marked(from)).toBe("Heat (1995)/[Heat (1995) - 4K.mkv]");
  expect(marked(to)).toBe("Heat (1995)/[Heat (1995) - 4K HDR.mkv]");
});
