import type { Layout, QualityKey } from "./order";
import { listed } from "./words";

/**
 * Why a release may or may not take a cell, as Judge in internal/targets
 * decides it for quality alone: vacant and outside take any accepted
 * quality, upgrade ranks above the held file, and final, tied and lower
 * are Judge's not_an_upgrade.
 */
export type Why = "vacant" | "outside" | "upgrade" | "not_in_target" | "final" | "tied" | "lower";

export type Move = { readonly from: QualityKey | null; readonly to: QualityKey };

export type Judged = { readonly allowed: boolean; readonly why: Why };

const rankIn = (tiers: Layout["tiers"], key: QualityKey): number | undefined => {
  const at = tiers.findIndex((tier) => tier.includes(key));
  return at === -1 ? undefined : at;
};

export function judgeMove({ tiers, cutoff }: Layout, { from, to }: Move): Judged {
  const rank = rankIn(tiers, to);
  if (rank === undefined) {
    return { allowed: false, why: "not_in_target" };
  }
  if (from === null) {
    return { allowed: true, why: "vacant" };
  }
  const held = rankIn(tiers, from);
  if (held === undefined) {
    return { allowed: true, why: "outside" };
  }
  if (held <= cutoff) {
    return { allowed: false, why: "final" };
  }
  if (rank === held) {
    return { allowed: false, why: "tied" };
  }
  return rank < held ? { allowed: true, why: "upgrade" } : { allowed: false, why: "lower" };
}

export type Sample = Judged & {
  readonly from: QualityKey;
  readonly to: QualityKey;
  readonly reason: string;
};

type Name = (key: QualityKey) => string;

const either = (names: readonly string[]): string => listed(names, "or");

const reasons: { readonly [W in Why]: (from: string, to: string) => string } = {
  upgrade: (from, to) => `${to} ranks above ${from}.`,
  outside: (from, to) => `${to} ranks above ${from}.`,
  vacant: (from, to) => `${to} ranks above ${from}.`,
  not_in_target: (_from, to) => `This Target doesn't accept ${to}.`,
  final: (from) => `${from} has reached the cutoff, so nothing replaces it.`,
  tied: (from, to) => `${from} and ${to} are tied.`,
  lower: (from, to) => `${to} ranks below ${from}.`,
};

/** The moves worth showing for a layout: an upgrade, a tie, the cutoff, a step down and a quality left out. */
function sampleMoves(layout: Layout, table: readonly QualityKey[]): [QualityKey, QualityKey][] {
  const { tiers, cutoff } = layout;
  const first = (at: number) => tiers[at]?.[0];
  const bottom = tiers.length - 1;
  const ties = [...tiers.keys()].filter((at) => (tiers[at]?.length ?? 0) > 1);
  const tied = ties.find((at) => at > cutoff) ?? ties[0];
  const accepted = new Set(tiers.flat());
  const left = table.toReversed().find((key) => key !== "cam" && !accepted.has(key));
  const pairs = [
    [first(bottom), first(bottom - 1)],
    [first(bottom), first(0)],
    [first(cutoff), cutoff > 0 ? first(0) : tiers[0]?.[1]],
    tied === undefined ? [] : [tiers[tied]?.[0], tiers[tied]?.[1]],
    [first(cutoff + 1), first(cutoff + 2)],
    [first(bottom), left],
  ];
  const seen = new Set<string>();
  return pairs.flatMap(([from, to]) => {
    if (from === undefined || to === undefined || from === to || seen.has(`${from}>${to}`)) {
      return [];
    }
    seen.add(`${from}>${to}`);
    return [[from, to]];
  });
}

/**
 * The sentence under the Qualities list and up to six sample moves, such as
 * "Grabs WEBRip-720p or better. Upgrades until Bluray-1080p, then stops."
 * `table` is every quality, low to high, for a sample of one left out.
 */
export function upgradeSentence(
  layout: Layout,
  table: readonly QualityKey[],
  name: Name,
): { sentence: string; samples: Sample[] } {
  const { tiers, cutoff } = layout;
  const bottom = tiers.at(-1);
  if (bottom === undefined) {
    return { sentence: "Accepts no quality yet, so it grabs nothing.", samples: [] };
  }
  const grabs = `Grabs ${either(bottom.map((key) => name(key)))}${tiers.length > 1 ? " or better" : ""}.`;
  const stops =
    cutoff >= tiers.length - 1
      ? "Keeps the first file it grabs."
      : `Upgrades until ${either((tiers[cutoff] ?? []).map((key) => name(key)))}, then stops.`;
  const samples = sampleMoves(layout, table).map(([from, to]): Sample => {
    const { allowed, why } = judgeMove(layout, { from, to });
    return { allowed, why, from, to, reason: reasons[why](name(from), name(to)) };
  });
  return { sentence: `${grabs} ${stops}`, samples };
}
