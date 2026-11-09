import { expect, test } from "vitest";

import type { Layout, QualityKey } from "./order";
import { type Rules, upgradeSentence } from "./upgrade-sentence";

const names: Partial<Record<QualityKey, string>> = {
  "webdl-1080p": "WEBDL-1080p",
  "webrip-1080p": "WEBRip-1080p",
  "bluray-1080p": "Bluray-1080p",
};

const name = (quality: QualityKey) => names[quality] ?? quality;

const table: readonly QualityKey[] = ["webrip-1080p", "webdl-1080p", "bluray-1080p"];

const scoring = (layout: Layout, upgradeUntil: number): Rules => ({
  ...layout,
  scores: { minimum: 0, upgradeUntil, increment: 1 },
  propers: "prefer_and_upgrade",
});

const twoTiers: Layout = { tiers: [["bluray-1080p"], ["webdl-1080p", "webrip-1080p"]], cutoff: 0 };

test("an upgrade-until score joins the cutoff in the sentence", () => {
  expect(upgradeSentence(scoring(twoTiers, 50), table, name).sentence).toBe(
    "Grabs WEBDL-1080p or WEBRip-1080p or better. Upgrades until Bluray-1080p and a score of 50, then stops.",
  );
});

test("with the cutoff at the bottom, an upgrade-until score alone upgrades", () => {
  expect(upgradeSentence(scoring({ ...twoTiers, cutoff: 1 }, -10), table, name).sentence).toBe(
    "Grabs WEBDL-1080p or WEBRip-1080p or better. Upgrades until a score of -10, then stops.",
  );
  expect(upgradeSentence(scoring({ ...twoTiers, cutoff: 1 }, 0), table, name).sentence).toBe(
    "Grabs WEBDL-1080p or WEBRip-1080p or better. Keeps the first file it grabs.",
  );
});
