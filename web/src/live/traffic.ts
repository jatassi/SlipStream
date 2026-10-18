import type { NodesTraffic } from "../api/gen/model";

// The last two nodes.traffic samples, outside React Query: a sample a second
// would otherwise re-render whatever reads it. The Diagram interpolates
// between the two.

export type TrafficPair = readonly [NodesTraffic | undefined, NodesTraffic | undefined];

let pair: TrafficPair = [undefined, undefined];
const listeners = new Set<() => void>();

/** Takes a sample in: it becomes the latest, and the latest becomes the one before. */
export function takeTraffic(sample: NodesTraffic): void {
  pair = [pair[1], sample];
  for (const listener of listeners) {
    listener();
  }
}

/** The sample before the latest, and the latest. */
export const trafficPair = (): TrafficPair => pair;

export function subscribeTraffic(listener: () => void): () => void {
  listeners.add(listener);
  return () => {
    listeners.delete(listener);
  };
}
