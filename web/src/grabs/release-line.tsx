import { Link } from "@tanstack/react-router";

import type {
  GrabsListedRelease,
  GrabsListedReleaseOutcome,
  GrabsWordedReason,
} from "../api/gen/model";
import { countWords } from "../format";
import { Data } from "../ui/data";
import { type Reference, lostAtSentence } from "./lost-at-sentence";

const list = new Intl.ListFormat("en", { type: "conjunction" });

/** What a run's Release lines read beyond each Release. */
export type RunNames = {
  readonly targets: ReadonlyMap<string, string>;
  readonly formats: ReadonlyMap<string, string>;
  /** The Targets the run's Grab was grabbed for, by name. */
  readonly grabbedFor: readonly string[];
  /** The Indexer that listed each position, for naming the Release a copy copies. */
  readonly indexerAt: ReadonlyMap<number, string>;
};

const seeders = { one: "seeder", many: "seeders" };

const lost = (release: GrabsListedRelease, names: RunNames, reference: Reference) =>
  release.lostAt === null
    ? undefined
    : lostAtSentence(
        release.lostAt,
        reference,
        release.lostAt.targetId === null ? undefined : names.targets.get(release.lostAt.targetId),
      );

/** A note, such as size_not_checked, says something about a Release without refusing it. */
const isNote = (reason: GrabsWordedReason): boolean => reason.code === "size_not_checked";

/**
 * What a refused Release's reasons add up to: a hard refusal keeps every
 * search from grabbing it, and soft ones keep only an automatic search.
 */
function refusalWords(reasons: readonly GrabsWordedReason[]): string {
  if (reasons.some((reason) => !reason.soft && !isNote(reason))) {
    return "No search can grab it.";
  }
  if (reasons.some((reason) => reason.soft)) {
    return "An automatic search skips it. A Manual search can grab it.";
  }
  return "No Target this search was for can take it.";
}

const outcomes: Record<
  GrabsListedReleaseOutcome,
  (release: GrabsListedRelease, names: RunNames) => string
> = {
  grabbed: (_, { grabbedFor }) =>
    grabbedFor.length === 0 ? "This search grabbed it." : `Grabbed for ${list.format(grabbedFor)}.`,
  not_taken: () =>
    "It ranked above the grabbed Release. Its Indexer stopped, reached its limit or dropped it.",
  outranked: (release, names) =>
    lost(release, names, "grabbed") ?? "It ties the grabbed Release at every step.",
  kept: (release, names) =>
    lost(release, names, "first") ?? "It ranked first. The search grabbed nothing.",
  refused: (release) => refusalWords(release.reasons),
};

/** Its outcome in words: what it was grabbed for, where it lost, or what refused it. */
export const outcomeWords = (release: GrabsListedRelease, names: RunNames): string =>
  outcomes[release.outcome](release, names);

function copyWords(release: GrabsListedRelease, names: RunNames): string {
  const original = release.copyOf === null ? undefined : names.indexerAt.get(release.copyOf);
  return original === undefined ? "" : ` It's the same Release as the one ${original} listed.`;
}

function scoreWords(release: GrabsListedRelease, names: RunNames): string | undefined {
  if (release.formatIds.length === 0) {
    return undefined;
  }
  const matched = release.formatIds.map((id) => names.formats.get(id) ?? "a deleted Custom Format");
  const scores = release.scores.map(
    ({ targetId, score }) => `${score} at ${names.targets.get(targetId) ?? "a deleted Target"}`,
  );
  return `Matches ${list.format(matched)}.${scores.length === 0 ? "" : ` Scores ${list.format(scores)}.`}`;
}

/**
 * One Release a run saw: the facts in words, the raw title under them, its
 * outcome in a sentence, each reason in words, and a link that opens the
 * tester with its title.
 */
export function ReleaseLine({ release, names }: { release: GrabsListedRelease; names: RunNames }) {
  const scores = scoreWords(release, names);
  return (
    <li className="flex min-w-0 flex-col gap-1 py-3">
      <span className="text-ink-1">{release.words}</span>
      <span className="text-meta break-all text-ink-3">
        <Data>{release.title}</Data>
      </span>
      <span className="text-meta text-ink-3">
        {release.indexerName} · {countWords(release.seeders, seeders)}
      </span>
      <p className="text-ink-2">
        {outcomeWords(release, names)}
        {copyWords(release, names)}
      </p>
      {release.reasons.length > 0 && (
        <ul aria-label="Reasons" className="flex list-disc flex-col gap-0.5 pl-5 text-ink-2">
          {release.reasons.map((reason, at) => (
            // oxlint-disable-next-line react/no-array-index-key -- a reason has no id, and the list never reorders
            <li key={at}>{reason.words}</li>
          ))}
        </ul>
      )}
      {scores !== undefined && <p className="text-meta text-ink-3">{scores}</p>}
      <Link
        to="/settings/$section"
        params={{ section: "custom-formats" }}
        search={{ release: release.title }}
        className="self-start rounded-item font-medium text-ink-2 underline-offset-4 hover:text-ink-1 hover:underline"
      >
        Open in tester
      </Link>
    </li>
  );
}
