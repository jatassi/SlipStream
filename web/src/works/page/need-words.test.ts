import { expect, test } from "vitest";

import type { FilesFailedNeed, WorksItem } from "../../api/gen/model";
import type { FailedGrab } from "../../grabs/stages";
import { anthologies, longShelf } from "../../test/anthology";
import { imported } from "../../test/grabs";
import { cinema, orchard, orchardItemId } from "../../test/library";
import { failedKey, failedWords, mergedFailures, upgradeWords } from "./need-words";

const failed: FailedGrab = {
  ...imported,
  stage: "failed",
  failure: { stage: "importing", code: "import_failed", message: "permission denied" },
};

const names: Record<string, string> = { uhd: "2160p HDR", hd: "1080p", sd: "720p" };
const naming = {
  type: cinema,
  work: orchard,
  targetName: (id: string) => names[id] ?? id,
  cutName: (key: string) => (key === "directors-cut" ? "Director's Cut" : key),
};

const need: FilesFailedNeed = {
  kind: "failed",
  itemIds: [orchardItemId],
  edition: "directors-cut",
  targetId: "uhd",
  waiting: ["hd", "sd"],
  grabId: failed.id,
};

test("a failure names its cell, its stage and the cells that wait on it", () => {
  expect(failedWords(naming, need, failed)).toBe(
    "Director's Cut 2160p HDR failed at Import · 1080p and 720p wait on it",
  );
  expect(failedWords(naming, { ...need, edition: "any", waiting: ["hd"] }, failed)).toBe(
    "2160p HDR failed at Import · 1080p waits on it",
  );
});

test("an upgrade names every Target it would better", () => {
  expect(
    upgradeWords(naming, {
      kind: "upgrade",
      itemIds: [orchardItemId],
      edition: "any",
      upgrades: ["uhd", "hd"],
    }),
  ).toBe("Upgrade available for 2160p HDR and 1080p");
});

const [, , third, fourth, fifth, sixth, seventh] = longShelf.items;
const shelfNaming = { ...naming, type: anthologies, work: longShelf };
const failedAt = (stage: "downloading" | "importing", id: string): FailedGrab => ({
  ...failed,
  id,
  failure: { ...failed.failure, stage },
});
const entry = (
  item: WorksItem | undefined,
  grab: FailedGrab,
  over: Partial<FilesFailedNeed> = {},
) => ({
  need: {
    ...need,
    itemIds: [item?.id ?? ""],
    edition: "any",
    waiting: [],
    grabId: grab.id,
    ...over,
  },
  grab,
});
const ids = ["a1", "a2", "a3", "a4", "a5"].map((n) => `00000000-0000-4000-8000-0000000000${n}`);

test("consecutive Items that failed alike merge into one run named by full codes", () => {
  const runs = mergedFailures(longShelf.items, [
    entry(third, failedAt("downloading", ids[0] ?? "")),
    entry(fourth, failedAt("downloading", ids[1] ?? "")),
    entry(fifth, failedAt("downloading", ids[2] ?? "")),
    entry(sixth, failedAt("downloading", ids[3] ?? "")),
  ]);
  expect(runs).toHaveLength(1);
  const [run] = runs;
  expect(run?.need.itemIds).toEqual([third?.id, fourth?.id, fifth?.id, sixth?.id]);
  expect(run && failedWords(shelfNaming, run.need, run.grab)).toBe(
    "V01S03 to V01S06 2160p HDR failed at Downloading",
  );
});

test("a gap, another stage, Target or waiting list starts a new run", () => {
  const at = (n: number) => failedAt("downloading", ids[n] ?? "");
  const runs = mergedFailures(longShelf.items, [
    entry(third, at(0)),
    entry(fifth, at(1)),
    entry(sixth, failedAt("importing", ids[2] ?? "")),
    entry(seventh, at(3), { targetId: "hd" }),
    entry(seventh, at(4), { waiting: ["sd"] }),
  ]);
  expect(runs.map((run) => failedWords(shelfNaming, run.need, run.grab))).toEqual([
    "V01S03 2160p HDR failed at Downloading",
    "V01S05 2160p HDR failed at Downloading",
    "V01S06 2160p HDR failed at Import",
    "V01S07 1080p failed at Downloading",
    "V01S07 2160p HDR failed at Downloading · 720p waits on it",
  ]);
  expect(new Set(runs.map((run) => failedKey(run))).size).toBe(runs.length);
});
