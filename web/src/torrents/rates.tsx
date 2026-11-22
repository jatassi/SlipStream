import { useSyncExternalStore } from "react";

import type { NodesTraffic } from "../api/gen/model";
import { subscribeTraffic, trafficPair } from "../live/traffic";
import { Data } from "../ui/data";
import { rateWords } from "./model";

/** The latest traffic sample, undefined until one arrives. */
export function useLatestTraffic(): NodesTraffic | undefined {
  return useSyncExternalStore(subscribeTraffic, trafficPair)[1];
}

export type Pace = { readonly downBps: number; readonly upBps: number };

/** Every Node's download and upload rates summed. */
export const totalOf = ({ nodes }: NodesTraffic): Pace => ({
  downBps: nodes.reduce((sum, node) => sum + node.downBps, 0),
  upBps: nodes.reduce((sum, node) => sum + node.upBps, 0),
});

/** Whether any byte moves, since a sample at rest reads only dashes. */
export const moving = ({ downBps, upBps }: Pace): boolean => downBps > 0 || upBps > 0;

/** "↓ 42.1 MB/s · ↑ 3.2 MB/s" */
export function Rates({ pace: { downBps, upBps } }: { pace: Pace }) {
  return (
    <span className="whitespace-nowrap">
      <Data>
        <span aria-label="Down">↓</span> {rateWords(downBps)}
      </Data>
      <span aria-hidden="true" className="px-1.5 text-ink-4">
        ·
      </span>
      <Data>
        <span aria-label="Up">↑</span> {rateWords(upBps)}
      </Data>
    </span>
  );
}
