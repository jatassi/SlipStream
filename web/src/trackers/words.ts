import type { TrackersGoal, TrackersRule, TrackersTrackerItemPrivacy } from "../api/gen/model";
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
  rule.source === "admin" ? "Set by you" : `From ${name}'s releases`;

/** What a Tracker with no rule in force says instead of its rule. */
export function noRuleSentence(name: string, privacy: TrackersTrackerItemPrivacy): string {
  if (privacy === "public") {
    return `${name} is public and has no seeding rule. Its torrents seed until your seeding goal is met.`;
  }
  return `${name}'s releases don't include a seeding rule. SlipStream keeps seeding its private torrents until you set one.`;
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
    return "Each torrent seeds until you remove it.";
  }
  return `Each torrent seeds until it reaches ${reaches(goal)}. ${actionSentences[goal.action]}`;
}
