import type {
  TrackersCounts,
  TrackersGoal,
  TrackersRule,
  TrackersTrackerItemPrivacy,
} from "../api/gen/model";
import { durationWords } from "../format";

type Action = NonNullable<TrackersGoal["action"]>;

export const privacyLabels: { readonly [P in TrackersTrackerItemPrivacy]: string } = {
  private: "Private",
  "semi-private": "Semi-private",
  public: "Public",
  unknown: "Not known yet",
};

export const ratioWords = (ratio: number): string =>
  Number.isInteger(ratio) ? ratio.toFixed(1) : String(ratio);

const seedingWords = (seconds: number): string => `${durationWords(seconds)} of seeding`;

const ruleParts: {
  readonly [C in TrackersRule["clears"]]: (rule: TrackersRule) => string;
} = {
  time: (rule) => `${seedingWords(rule.seedSeconds ?? 0)} for each torrent. Ratio doesn't count.`,
  ratio: (rule) =>
    `a ratio of ${ratioWords(rule.ratio ?? 0)} for each torrent. Seeding time doesn't count.`,
  either: (rule) =>
    `a ratio of ${ratioWords(rule.ratio ?? 0)} or ${seedingWords(rule.seedSeconds ?? 0)} for each torrent.`,
  both: (rule) =>
    `a ratio of ${ratioWords(rule.ratio ?? 0)} and ${seedingWords(rule.seedSeconds ?? 0)} for each torrent.`,
};

/** "seedpool requires 10 days of seeding for each torrent. Ratio doesn't count." */
export const ruleSentence = (name: string, rule: TrackersRule): string =>
  `${name} requires ${ruleParts[rule.clears](rule)}`;

export const ruleSource = (name: string, rule: TrackersRule): string =>
  rule.source === "admin" ? "Set by you" : `From ${name}'s Releases`;

/** What a Tracker with no rule in force says instead of its rule. */
export function noRuleSentence(name: string, privacy: TrackersTrackerItemPrivacy): string {
  if (privacy === "public") {
    return `${name} is public, so its torrents follow your seeding goal alone.`;
  }
  return `${name}'s Releases don't state a seeding rule, so SlipStream never stops its private torrents. Set ${name}'s rule to let its goal run.`;
}

const actionSentences: { readonly [A in Action]: string } = {
  pause: "Then SlipStream pauses it.",
  remove: "Then SlipStream removes it.",
  "remove-delete": "Then SlipStream removes it and deletes its download copy.",
};

export const actionLabels: { readonly [A in Action]: string } = {
  pause: "Pause",
  remove: "Remove",
  "remove-delete": "Remove and delete the download copy",
};

function reaches(goal: TrackersGoal): string {
  const ratio = goal.ratio === null ? undefined : `a ratio of ${ratioWords(goal.ratio)}`;
  const time = goal.seedSeconds === null ? undefined : seedingWords(goal.seedSeconds);
  if (ratio === undefined || time === undefined) {
    return ratio ?? time ?? "";
  }
  return goal.combine === "both"
    ? `${ratio} and ${time}`
    : `${ratio} or ${time}, whichever comes first`;
}

/** A goal in sentences: how far each torrent seeds, then what happens to it. */
export function goalSentence(goal: TrackersGoal): string {
  if (goal.combine === "forever" || goal.action === null) {
    return "Each torrent seeds with no end until you remove it.";
  }
  return `Each torrent seeds until it reaches ${reaches(goal)}. ${actionSentences[goal.action]}`;
}

const plural = (count: number, one: string, many: string): string =>
  `${count} ${count === 1 ? one : many}`;

/** "214 torrents are seeding. 3 still owe seedpool seeding, and 1 of those isn't seeding." */
export function standingSentence(name: string, counts: TrackersCounts): string {
  if (counts.torrents === 0) {
    return `SlipStream has no torrents from ${name}.`;
  }
  const seeding =
    counts.seeding === 0
      ? counts.torrents === 1
        ? "Its one torrent isn't seeding."
        : `None of its ${counts.torrents} torrents are seeding.`
      : `${plural(counts.seeding, "torrent is", "torrents are")} seeding.`;
  if (counts.owing === 0) {
    return `${seeding} None of them owe ${name} seeding.`;
  }
  const owing = `${counts.owing} still ${counts.owing === 1 ? "owes" : "owe"} ${name} seeding`;
  if (counts.atRisk === 0) {
    return `${seeding} ${owing}.`;
  }
  const risk = counts.atRisk === 1 ? "isn't" : "aren't";
  return `${seeding} ${owing}, and ${counts.atRisk} of those ${risk} seeding.`;
}
