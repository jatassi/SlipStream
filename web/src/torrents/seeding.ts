import type {
  DownloadclientsOwes,
  DownloadclientsSeedingView,
  DownloadclientsSeedingViewAction,
  DownloadclientsSeedingViewEndless,
} from "../api/gen/model";
import { bytes, spanWords } from "../format";
import type { Lane } from "../ui/lane-meter";

export type Seeding = DownloadclientsSeedingView;

/** A ratio as the rules state one: "1.0", "1.25". */
export const ratioWords = (ratio: number) => ratio.toFixed(2).replace(/(\.\d)0$/u, "$1");

export const lanesOf = ({ ratio, time }: Seeding): readonly [Lane, Lane] => [
  { label: "Ratio", ...ratio, words: ratioWords },
  { label: "Seeding time", ...time, words: spanWords },
];

/** "3 days 4 h of seeding or 1.2 GB of upload", from what the rule still needs. */
export function owedWords(
  { seconds, bytes: upload }: DownloadclientsOwes,
  clears: Seeding["clears"],
) {
  const parts = [
    ...(seconds > 0 ? [`${spanWords(seconds)} of seeding`] : []),
    ...(upload > 0 ? [`${bytes(upload)} of upload`] : []),
  ];
  return parts.join(clears === "either" ? " or " : " and ");
}

/** What the row says beside its lanes. */
export function seedLine(seeding: Seeding): string {
  if (seeding.owes !== undefined) {
    return `Owes ${owedWords(seeding.owes, seeding.clears)}`;
  }
  return seeding.endless === "none" ? "" : "Seeds with no end";
}

const ruleTime = ({ time }: Seeding) => spanWords((time.rule ?? 0) - (time.margin ?? 0));

export function ruleSentence(tracker: string, seeding: Seeding): string | undefined {
  const ratio = `a ratio of ${ratioWords(seeding.ratio.rule ?? 0)}`;
  const time = `${ruleTime(seeding)} of seeding`;
  switch (seeding.clears) {
    case "none":
      return undefined;
    case "time":
      return `${tracker} requires ${time} for each torrent. Ratio doesn't count.`;
    case "ratio":
      return `${tracker} requires ${ratio} for each torrent. Seeding time doesn't count.`;
    case "either":
      return `${tracker} requires ${ratio} or ${time} for each torrent.`;
    case "both":
      return `${tracker} requires ${ratio} and ${time} for each torrent.`;
    default:
      return seeding.clears satisfies never;
  }
}

export function marginSentence(tracker: string, { time }: Seeding): string | undefined {
  return time.margin === undefined || time.margin === 0
    ? undefined
    : `This includes ${spanWords(time.margin)} past ${tracker}'s rule.`;
}

const thenWords: { readonly [A in DownloadclientsSeedingViewAction]: string } = {
  pause: "SlipStream pauses it",
  remove: "SlipStream removes it",
  "remove-delete": "SlipStream removes it and deletes its download copy",
};

/** The goal and what happens once it's met, or undefined while its goal never acts. */
export function goalSentence({
  ratio,
  time,
  combine,
  action,
  endless,
}: Seeding): string | undefined {
  if (endless !== "none" || action === undefined) {
    return undefined;
  }
  const parts = [
    ...(ratio.goal === undefined ? [] : [`a ratio of ${ratioWords(ratio.goal)}`]),
    ...(time.goal === undefined ? [] : [`${spanWords(time.goal)} of seeding`]),
  ];
  const goal =
    parts.length === 2 && combine === "first"
      ? `${parts.join(" or ")}, whichever comes first`
      : parts.join(" and ");
  return `Its goal is ${goal}. Once it's met, ${thenWords[action]}.`;
}

const endlessWords: {
  readonly [E in Exclude<DownloadclientsSeedingViewEndless, "none">]: (tracker: string) => string;
} = {
  forever: () => "Its goal is Forever, so it seeds with no end.",
  resumed: () => "It seeds with no end, because you resumed it after its goal was met.",
  "no-tracker": () =>
    "This torrent has no known Tracker, so SlipStream never stops it. Remove it yourself once its Tracker's rule is met.",
  "no-rule": (tracker) =>
    `${tracker}'s Releases don't state a seeding rule, so SlipStream never stops this torrent. Set ${tracker}'s rule so that its goal can act.`,
  partial: (tracker) =>
    `Only some of its files are selected, so ${tracker} never counts it as seeding and SlipStream never stops it.`,
};

/** Why its goal never acts on it, or undefined while its goal can. */
export const endlessSentence = (tracker: string, { endless }: Seeding) =>
  endless === "none" ? undefined : endlessWords[endless](tracker);
