import type { NodesNodeStanding } from "../api/gen/model";
import { type Clock, sinceWords } from "../format";

export type Away = Exclude<NodesNodeStanding, "in-step">;

const awayWords: { readonly [S in Away]: string } = {
  reconnecting: "reconnecting",
  offline: "offline",
  lapsed: "offline",
  mismatched: "on another version",
};

/** "reconnecting" through the blip, then "offline since 21:31". */
export const awayLine = (standing: Away, since: string, clock: Clock): string =>
  standing === "reconnecting"
    ? awayWords.reconnecting
    : `${awayWords[standing]} since ${sinceWords(since, clock)}`;

/** Work that waits for its Node: "Waiting for seedbox · offline since 21:31". */
export const waitingLine = (node: string, standing: Away, since: string, clock: Clock): string =>
  `Waiting for ${node} · ${awayLine(standing, since, clock)}`;
