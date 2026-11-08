import { expect, test } from "vitest";

import type { GrabsReason, GrabsReasonCode } from "../api/gen/model";
import { type ReasonNames, reasonWords, refusalWords } from "./reason-words";

const hd = "6f1d2c3b-4a5e-4f6a-8b7c-9d0e1f2a3b4c";
const gone = "0e9d8c7b-6a5f-4e3d-9c2b-1a0f9e8d7c6b";

const names: ReasonNames = {
  targets: new Map([
    [hd, { name: "1080p", scores: { formats: [], increment: 1, minimum: 20, upgradeUntil: 0 } }],
  ]),
  qualities: new Map([["webdl-720p", "WEBDL-720p"]]),
};

const reason = (code: GrabsReasonCode, more: Partial<GrabsReason> = {}): GrabsReason => ({
  code,
  soft: false,
  ...more,
});

test.each<[GrabsReason, string]>([
  [reason("wrong_work"), "An id in the name belongs to another Work."],
  [reason("wrong_item", { value: "S01E02" }), "The Work has no Item S01E02."],
  [reason("wrong_item"), "It names none of the Work's Items."],
  [reason("covers_more", { soft: true, value: "S01" }), "It holds more than one Item."],
  [
    reason("wrong_title", { soft: true, value: "Heat (1986)" }),
    "Title doesn't match: Heat (1986).",
  ],
  [
    reason("cut_not_wanted", { soft: true, targetId: hd, value: "Extended" }),
    "The Work doesn't want the Extended cut at 1080p.",
  ],
  [
    reason("quality_not_in_target", { targetId: hd, value: "webdl-720p" }),
    "1080p doesn't accept WEBDL-720p.",
  ],
  [
    reason("quality_not_in_target", { targetId: hd }),
    "1080p doesn't accept a release with no Quality.",
  ],
  [
    reason("filter", { targetId: hd, attribute: "video_codec", value: "x265" }),
    "It fails the video codec filter of 1080p.",
  ],
  [reason("size_below", { soft: true, targetId: hd }), "It's smaller than 1080p allows."],
  [reason("size_above", { soft: true, targetId: hd }), "It's larger than 1080p allows."],
  [
    reason("size_not_checked", { targetId: hd }),
    "Its size isn't checked at 1080p because the name gives no size or the runtime is unknown.",
  ],
  [
    reason("score_below_minimum", { soft: true, targetId: hd, value: "10" }),
    "Score 10 is below the minimum of 20 at 1080p.",
  ],
  [
    reason("not_an_upgrade", { soft: true, targetId: hd }),
    "It isn't an upgrade of the file at 1080p.",
  ],
  [
    reason("repack_other_group", { soft: true, targetId: hd }),
    "This repack is from another release group than the file it would replace.",
  ],
])("%j reads %j", (given, words) => {
  expect(reasonWords(given, names)).toBe(words);
});

test("a reason at a Target deleted since the run names it as deleted", () => {
  expect(
    reasonWords(reason("quality_not_in_target", { targetId: gone, value: "webdl-720p" }), names),
  ).toBe("A deleted Target doesn't accept WEBDL-720p.");
  expect(
    reasonWords(reason("score_below_minimum", { soft: true, targetId: gone, value: "10" }), names),
  ).toBe("Score 10 is below the minimum of a deleted Target.");
});

test("one hard refusal keeps every search from grabbing a Release", () => {
  expect(
    refusalWords([
      reason("wrong_title", { soft: true, value: "Heat (1986)" }),
      reason("quality_not_in_target", { targetId: hd, value: "webdl-720p" }),
    ]),
  ).toBe("No search can grab it.");
});

test("soft refusals alone keep only an automatic search from grabbing it", () => {
  expect(
    refusalWords([
      reason("wrong_title", { soft: true, value: "Heat (1986)" }),
      reason("size_not_checked", { targetId: hd }),
    ]),
  ).toBe("An automatic search skips it. A Manual search can grab it.");
});

test("a note refuses nothing, so a Release with only notes was for no Target the search wanted", () => {
  expect(refusalWords([reason("size_not_checked", { targetId: hd })])).toBe(
    "No Target this search was for can take it.",
  );
  expect(refusalWords([])).toBe("No Target this search was for can take it.");
});
