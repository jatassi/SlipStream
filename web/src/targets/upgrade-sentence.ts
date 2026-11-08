import type { TargetsScores, TargetsTargetPropers } from "../api/gen/model";
import { type Layout, type QualityKey, cam } from "./order";
import { listed } from "./words";

/** What Judge in internal/targets reads of a Target: its layout, score settings and propers setting. */
export type Rules = Layout & {
  readonly scores: Pick<TargetsScores, "minimum" | "upgradeUntil" | "increment">;
  readonly propers: TargetsTargetPropers;
};

/**
 * A release, or the file it would replace. The version is 1 unless a
 * proper or repack raised it, real counts REAL, and a null group is
 * unknown.
 */
export type Side = {
  readonly quality: QualityKey;
  readonly version: number;
  readonly real: number;
  readonly repack: boolean;
  readonly group: string | null;
  readonly score: number;
};

export type Move = { readonly from: Side | null; readonly to: Side };

/**
 * Why a release may or may not take a cell, as Judge decides it. Each but
 * not_in_target is one of Judge's numbered upgrade rules.
 */
export type Why =
  | "not_in_target"
  | "vacant"
  | "outside"
  | "proper"
  | "other_group"
  | "upgrade"
  | "final"
  | "lower"
  | "older"
  | "tied"
  | "gain"
  | "short";

type Judged = {
  readonly allowed: boolean;
  readonly why: Why;
  readonly rule: number;
  readonly belowMinimum: boolean;
};

/** A release against the file filling its cell, as the upgrade rules read it, mirroring Go's move. */
type Compared = {
  readonly vacant: boolean;
  readonly outside: boolean;
  /** +1 when the release's tier is better than the file's, -1 when worse. */
  readonly tier: number;
  /** The release's revision against the file's within one Quality, 0 across Qualities. */
  readonly revision: number;
  readonly heldFinal: boolean;
  readonly repack: boolean;
  readonly sameGroup: boolean;
  readonly propers: TargetsTargetPropers;
  readonly score: number;
  readonly held: number;
  readonly upgradeUntil: number;
  readonly increment: number;
};

const upgradeRules: readonly {
  readonly rule: number;
  readonly why: Why;
  readonly applies: (c: Compared) => boolean;
}[] = [
  { rule: 1, why: "vacant", applies: (c) => c.vacant },
  { rule: 1, why: "outside", applies: (c) => c.outside },
  {
    rule: 2,
    why: "other_group",
    applies: (c) =>
      c.revision > 0 && c.propers === "prefer_and_upgrade" && c.repack && !c.sameGroup,
  },
  { rule: 2, why: "proper", applies: (c) => c.revision > 0 && c.propers === "prefer_and_upgrade" },
  { rule: 3, why: "upgrade", applies: (c) => c.tier > 0 && !c.heldFinal },
  { rule: 4, why: "final", applies: (c) => c.tier > 0 },
  { rule: 5, why: "lower", applies: (c) => c.tier < 0 },
  { rule: 6, why: "older", applies: (c) => c.revision < 0 && c.propers !== "dont_prefer" },
  { rule: 7, why: "tied", applies: (c) => c.held >= c.upgradeUntil },
  { rule: 8, why: "gain", applies: (c) => c.score >= c.held + c.increment },
];

const otherwise = { rule: 8, why: "short" } as const;

const upgrading: ReadonlySet<Why> = new Set(["vacant", "outside", "proper", "upgrade", "gain"]);

const rankIn = (tiers: Layout["tiers"], key: QualityKey): number | undefined => {
  const at = tiers.findIndex((tier) => tier.includes(key));
  return at === -1 ? undefined : at;
};

const revisionOf = (a: Side, b: Side): number =>
  Math.sign(a.real - b.real) || Math.sign(Math.max(a.version, 1) - Math.max(b.version, 1));

const sameGroup = (a: string | null, b: string | null): boolean =>
  a !== null && b !== null && a.toLowerCase() === b.toLowerCase();

export function judgeMove(rules: Rules, { from, to }: Move): Judged {
  const { tiers, cutoff, scores, propers } = rules;
  const belowMinimum = to.score < scores.minimum;
  const rank = rankIn(tiers, to.quality);
  if (rank === undefined) {
    return { allowed: false, why: "not_in_target", rule: 0, belowMinimum };
  }
  const held = from === null ? undefined : rankIn(tiers, from.quality);
  const compared: Compared = {
    vacant: from === null,
    outside: held === undefined,
    tier: held === undefined ? 0 : Math.sign(held - rank),
    revision: from !== null && from.quality === to.quality ? revisionOf(to, from) : 0,
    heldFinal: held !== undefined && held <= cutoff,
    repack: to.repack,
    sameGroup: sameGroup(to.group, from?.group ?? null),
    propers,
    score: to.score,
    held: from?.score ?? 0,
    upgradeUntil: scores.upgradeUntil,
    increment: scores.increment,
  };
  const decided = upgradeRules.find((rule) => rule.applies(compared)) ?? otherwise;
  const allowed = upgrading.has(decided.why) && !belowMinimum;
  return { allowed, why: decided.why, rule: decided.rule, belowMinimum };
}

export type Sample = Judged & {
  readonly from: QualityKey;
  readonly to: QualityKey;
  readonly reason: string;
};

type Name = (key: QualityKey) => string;

const either = (names: readonly string[]): string => listed(names, "or");

type Said = {
  readonly from: string;
  readonly to: string;
  readonly move: { readonly from: Side; readonly to: Side };
  readonly rules: Rules;
};

const reasons: { readonly [W in Why]: (said: Said) => string } = {
  upgrade: ({ from, to }) => `${to} ranks above ${from}.`,
  outside: ({ from, to }) => `${to} ranks above ${from}.`,
  vacant: ({ from, to }) => `${to} ranks above ${from}.`,
  not_in_target: ({ to }) => `This Target doesn't accept ${to}.`,
  proper: ({ from }) => `This release is a newer revision of the ${from} file.`,
  other_group: () => "This repack is from another release group than the file it would replace.",
  final: ({ from }) => `${from} has reached the cutoff, so no better tier replaces it.`,
  lower: ({ from, to }) => `${to} ranks below ${from}.`,
  older: ({ from }) => `This release is an older revision of the ${from} file.`,
  tied: ({ from, to, move, rules }) =>
    move.from.score === 0 && rules.scores.upgradeUntil === 0
      ? `${from} and ${to} are tied.`
      : `${from} and ${to} are tied, and ${from} scores ${String(move.from.score)}, at or above this Target's upgrade-until score of ${String(rules.scores.upgradeUntil)}.`,
  gain: ({ from, to, move, rules }) =>
    `${to} scores ${String(move.to.score)} and ${from} scores ${String(move.from.score)}, a gain of at least ${String(rules.scores.increment)}.`,
  short: ({ from, to, move, rules }) =>
    `${to} scores ${String(move.to.score)}. It needs ${String(move.from.score + rules.scores.increment)} or more to replace ${from}.`,
};

/**
 * Judge's answer for a move over a held file, with its reasons in words,
 * such as "This repack is from another release group than the file it
 * would replace."
 */
export function judgeInWords(
  rules: Rules,
  move: { readonly from: Side; readonly to: Side },
  name: Name,
): Judged & { readonly reason: string } {
  const judged = judgeMove(rules, move);
  const said = reasons[judged.why]({
    from: name(move.from.quality),
    to: name(move.to.quality),
    move,
    rules,
  });
  const reason = judged.belowMinimum
    ? `${said} Score ${String(move.to.score)} is below this Target's minimum of ${String(rules.scores.minimum)}.`
    : said;
  return { ...judged, reason };
}

/** A release or file of quality with no revision, group or score. */
const plain = (quality: QualityKey): Side => ({
  quality,
  version: 1,
  real: 0,
  repack: false,
  group: null,
  score: 0,
});

/** The moves worth showing for a layout: an upgrade, a tie, the cutoff, a step down and a quality left out. */
function sampleMoves(layout: Layout, table: readonly QualityKey[]): [QualityKey, QualityKey][] {
  const { tiers, cutoff } = layout;
  const first = (at: number) => tiers[at]?.[0];
  const bottom = tiers.length - 1;
  const ties = [...tiers.keys()].filter((at) => (tiers[at]?.length ?? 0) > 1);
  const tied = ties.find((at) => at > cutoff) ?? ties[0];
  const accepted = new Set(tiers.flat());
  const left = table.toReversed().find((key) => key !== cam && !accepted.has(key));
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
 * "Grabs WEBRip-720p or better. Upgrades until Bluray-1080p and a score of
 * 50, then stops."
 * `table` is every quality, low to high, for a sample of one left out.
 */
export function upgradeSentence(
  rules: Rules,
  table: readonly QualityKey[],
  name: Name,
): { sentence: string; samples: Sample[] } {
  const { tiers, cutoff, scores } = rules;
  const bottom = tiers.at(-1);
  if (bottom === undefined) {
    return { sentence: "Accepts no quality yet, so it grabs nothing.", samples: [] };
  }
  const grabs = `Grabs ${either(bottom.map((key) => name(key)))}${tiers.length > 1 ? " or better" : ""}.`;
  const score = scores.upgradeUntil === 0 ? [] : [`a score of ${String(scores.upgradeUntil)}`];
  const until = [
    ...(cutoff >= tiers.length - 1 ? [] : [either((tiers[cutoff] ?? []).map((key) => name(key)))]),
    ...score,
  ];
  const stops =
    until.length === 0
      ? "Keeps the first file it grabs."
      : `Upgrades until ${until.join(" and ")}, then stops.`;
  const samples = sampleMoves(rules, table).map(([from, to]): Sample =>
    Object.assign(judgeInWords(rules, { from: plain(from), to: plain(to) }, name), { from, to }),
  );
  return { sentence: `${grabs} ${stops}`, samples };
}
