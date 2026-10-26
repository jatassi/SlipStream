import { useSyncExternalStore } from "react";

import { subscribeTraffic, trafficPair } from "../../../../live/traffic";
import { Details } from "../../../../ui/details";
import { InspectorSection } from "../../../../ui/inspector";
import type { Landing, LinkRow, NodeRow } from "./view";

const megabits = (bps: number): string => `${((bps * 8) / 1e6).toFixed(1)} Mbit/s`;

/** The Node link to a Worker in the Inspector: what it carries, how quickly, and who dialed. */
export function LinkWords({
  landing,
  link,
  worker,
}: {
  landing: Landing;
  link: LinkRow;
  worker: NodeRow;
}) {
  const [, latest] = useSyncExternalStore(subscribeTraffic, trafficPair);
  const sample = latest?.links.find(({ b }) => b === link.worker);
  const dialer = link.dialer === "worker" ? worker.name : landing.controller;
  const listener = link.dialer === "worker" ? landing.controller : worker.name;
  return (
    <>
      <InspectorSection>
        {link.state === "up" ? (
          <>
            <p>
              {sample === undefined
                ? "The link is up. No traffic is reported yet."
                : `${worker.name} sends ${landing.controller} ${megabits(sample.baBps)} and receives ${megabits(sample.abBps)}. This includes the link and every Transfer between them.`}
            </p>
            {link.rttMs !== undefined && (
              <p className="mt-2 text-ink-3">Each round trip takes {link.rttMs} ms.</p>
            )}
          </>
        ) : (
          <p>
            {link.state === "mismatched"
              ? "The link is up, but the two Nodes run different versions."
              : "The link is down."}
          </p>
        )}
      </InspectorSection>
      {link.dialer !== undefined && (
        <Details summary={`${dialer} dialed ${listener}.`}>
          <p>The capsules glow brightest at the end that dialed.</p>
        </Details>
      )}
    </>
  );
}
