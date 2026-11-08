import { expect, test } from "vitest";

import type { GrabsLostAt, GrabsLostAtStep } from "../api/gen/model";
import { lostAtSentence } from "./lost-at-sentence";

const hd = "6f1d2c3b-4a5e-4f6a-8b7c-9d0e1f2a3b4c";
const at = (step: GrabsLostAtStep, own: string, ref: string): GrabsLostAt => ({
  step,
  own,
  ref,
  targetId: ["tier", "revision", "score"].includes(step) ? hd : null,
});

test.each<[GrabsLostAt, string]>([
  [
    at("want", "720p", "1080p"),
    "It would fill 720p. The grabbed Release would fill 1080p, which comes first in the Profile.",
  ],
  [
    at("tier", "WEBDL-1080p", "Bluray-1080p"),
    "Its quality is WEBDL-1080p. The grabbed Release is Bluray-1080p, a higher tier at 1080p.",
  ],
  [
    at("revision", "v1", "v2 REPACK"),
    "Its revision is v1. The grabbed Release is v2 REPACK, a newer revision at 1080p.",
  ],
  [at("score", "10", "35"), "Its score at 1080p is 10. The grabbed Release scores 35."],
  [
    at("priority", "30", "25"),
    "Its Indexer's priority is 30. The grabbed Release's is 25, and a lower number comes first.",
  ],
  [
    at("freeleech", "1", "0.5"),
    "Its download counts 100% toward the ratio. The grabbed Release's counts 50%.",
  ],
  [at("seeders", "1", "40"), "It has 1 seeder. The grabbed Release has 40 seeders."],
  [
    at("size", "4509715660", "9876543210"),
    "It's 4.2 GB. The grabbed Release is 9.2 GB, and the larger one comes first.",
  ],
  [
    at("title", "Heat.1995.B", "Heat.1995.A"),
    "It ties the grabbed Release at every other step. Its title comes later in alphabetical order.",
  ],
])("a Release that lost at %j to the grabbed one reads %j", (lostAt, sentence) => {
  expect(lostAtSentence(lostAt, "grabbed", lostAt.targetId === null ? undefined : "1080p")).toBe(
    sentence,
  );
});

test("a kept Release reads against the first-ranked Release, since the run grabbed none", () => {
  expect(lostAtSentence(at("score", "10", "35"), "first", "1080p")).toBe(
    "Its score at 1080p is 10. The first-ranked Release scores 35.",
  );
});

test("a step at a Target deleted since the run leaves the Target out", () => {
  expect(lostAtSentence({ ...at("score", "10", "35"), targetId: null }, "grabbed", undefined)).toBe(
    "Its score is 10. The grabbed Release scores 35.",
  );
});
