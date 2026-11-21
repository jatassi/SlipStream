import type { TrackersWindow, TrackersWindowState } from "../api/gen/model";
import type { Clock } from "../ui/clock";
import { whenWords } from "../ui/time-words";
import { owedWords } from "../torrents/seeding";

const clearsOf = (window: TrackersWindow) => window.rule?.clears ?? "both";

const at = (verb: string, iso: string | null, clock: Clock) =>
  iso === null ? verb : `${verb} ${whenWords(iso, "at", clock)}`;

/** What an open window still needs: "Still needs 3 days 4 h of seeding". */
export const owesWords = (window: TrackersWindow): string =>
  window.owes === null
    ? "No rule known"
    : `Still needs ${owedWords(window.owes, clearsOf(window))}`;

type EndedState = Exclude<TrackersWindowState, "owing">;

export type EndedWindow = TrackersWindow & { state: EndedState };

const endedWords: {
  readonly [S in EndedState]: (window: TrackersWindow, clock: Clock) => string;
} = {
  met: (window, clock) => at("Met", window.metAt, clock),
  missed: (window, clock) => at("Missed", window.closedAt, clock),
  released: (window, clock) => at("Closed", window.closedAt, clock),
};

/** A window as the history tells it: one grabbed again after its miss reads as that miss. */
export const asEnded = (window: TrackersWindow): EndedWindow =>
  window.state === "owing" ? { ...window, state: "missed" } : { ...window, state: window.state };

/** "Met at 2:32 PM", "Missed on Nov 3 at 9:10 AM". */
export const endedLine = (window: EndedWindow, clock: Clock): string =>
  endedWords[window.state](window, clock);

/** The sentences under a window: how it stands with its Tracker, and what happened to it. */
export function windowSentences(window: TrackersWindow, tracker: string, clock: Clock): string[] {
  const stopped =
    window.quietSince === null
      ? "It isn't seeding."
      : `It stopped seeding ${whenWords(window.quietSince, "at", clock)}.`;
  const risk = {
    yes: `${stopped} Seed it again before ${tracker} counts a hit and run.`,
    unknown: "Every Node that holds it is offline, so whether it's seeding isn't known.",
    no: undefined,
  }[window.atRisk];
  const regrabbed =
    window.regrabbedAt === null ||
    window.closedAt === null ||
    Date.parse(window.regrabbedAt) < Date.parse(window.closedAt)
      ? undefined
      : window.state === "missed"
        ? `It was grabbed again ${whenWords(window.regrabbedAt, "at", clock)}.`
        : `It was missed ${whenWords(window.closedAt, "at", clock)} and grabbed again ${whenWords(window.regrabbedAt, "at", clock)}.`;
  const owed =
    window.state === "missed" && window.owed !== null
      ? `It still needed ${owedWords(window.owed, clearsOf(window))} when its last torrent was removed.`
      : undefined;
  return [
    ...(window.state === "owing" && risk !== undefined ? [risk] : []),
    ...(owed === undefined ? [] : [owed]),
    ...(regrabbed === undefined ? [] : [regrabbed]),
  ];
}
