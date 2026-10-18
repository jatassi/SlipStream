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
                ? "Up, with no traffic reported yet."
                : `Carrying ${megabits(sample.baBps)} to ${landing.controller} and ${megabits(sample.abBps)} to ${worker.name}.`}
            </p>
            {link.rttMs !== undefined && (
              <p className="mt-2 text-ink-3">Each round trip takes {link.rttMs} ms.</p>
            )}
          </>
        ) : (
          <p>
            {link.state === "mismatched"
              ? "Up, but the two are on different versions."
              : "Down. Nothing is getting through."}
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
