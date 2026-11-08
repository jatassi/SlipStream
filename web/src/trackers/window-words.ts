import type { TrackersWindow, TrackersWindowState } from "../api/gen/model";
import { type Clock, sinceWords } from "../format";
import { owedWords } from "../torrents/seeding";

const clearsOf = (window: TrackersWindow) => window.rule?.clears ?? "both";

const at = (verb: string, iso: string | null, clock: Clock) =>
  iso === null ? verb : `${verb} at ${sinceWords(iso, clock)}`;

/** What an open window still needs: "Owes 3 days 4 h of seeding". */
export const owesWords = (window: TrackersWindow): string =>
  window.owes === null ? "No rule known" : `Owes ${owedWords(window.owes, clearsOf(window))}`;

const endedWords: {
  readonly [S in Exclude<TrackersWindowState, "owing">]: (
    window: TrackersWindow,
    clock: Clock,
  ) => string;
} = {
  met: (window, clock) => at("Met", window.metAt, clock),
  missed: (window, clock) => at("Missed", window.closedAt, clock),
  released: (window, clock) => at("Released", window.closedAt, clock),
};

/** "Met at 14:32", "Missed at Nov 3, 09:10". */
export const endedLine = (
  window: TrackersWindow & { state: Exclude<TrackersWindowState, "owing"> },
  clock: Clock,
): string => endedWords[window.state](window, clock);

/** The sentences under a window: how it stands with its Tracker, and what happened to it. */
export function windowSentences(window: TrackersWindow, tracker: string, clock: Clock): string[] {
  const stopped =
    window.quietSince === null
      ? "It isn't seeding."
      : `It stopped seeding at ${sinceWords(window.quietSince, clock)}.`;
  const risk = {
    yes: `${stopped} Seed it again before ${tracker} counts a hit and run.`,
    unknown: "Every Node that holds it is offline, so whether it's seeding isn't known.",
    no: undefined,
  }[window.atRisk];
  const regrabbed =
    window.regrabbedAt === null || window.closedAt === null
      ? undefined
      : `It was missed at ${sinceWords(window.closedAt, clock)} and grabbed again at ${sinceWords(window.regrabbedAt, clock)}.`;
  const owed =
    window.state === "missed" && window.owed !== null
      ? `It still owed ${owedWords(window.owed, clearsOf(window))} when its last torrent was removed.`
      : undefined;
  return [
    ...(window.state === "owing" && risk !== undefined ? [risk] : []),
    ...(regrabbed === undefined ? [] : [regrabbed]),
    ...(owed === undefined ? [] : [owed]),
  ];
}
