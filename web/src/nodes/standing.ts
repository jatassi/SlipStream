import type { NodesNodeStanding } from "../api/gen/model";
import type { Clock } from "../ui/clock";
import { whenWords } from "../ui/time-words";

export type Away = Exclude<NodesNodeStanding, "in-step">;

/** Each standing as a word inside a sentence, such as "seedbox is offline". */
export const awayWords: { readonly [S in Away]: string } = {
  reconnecting: "reconnecting",
  offline: "offline",
  lapsed: "offline",
  mismatched: "on another version",
};

/** "reconnecting" through the blip, then "offline since 9:31 PM". */
export const awayLine = (standing: Away, since: string, clock: Clock): string =>
  standing === "reconnecting"
    ? awayWords.reconnecting
    : `${awayWords[standing]} ${whenWords(since, "since", clock)}`;

/** Work that waits for its Node: "Waiting for seedbox · offline since 9:31 PM". */
export const waitingLine = (node: string, standing: Away, since: string, clock: Clock): string =>
  `Waiting for ${node} · ${awayLine(standing, since, clock)}`;
