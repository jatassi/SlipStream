import { expect, test } from "vitest";

import type { FilesFailedNeed } from "../../api/gen/model";
import type { FailedGrab } from "../../grabs/stages";
import { imported } from "../../test/grabs";
import { cinema, orchard, orchardItemId } from "../../test/library";
import { failedWords, upgradeWords } from "./need-words";

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
