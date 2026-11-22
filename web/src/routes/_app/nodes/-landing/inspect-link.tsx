import { Suspense, useSyncExternalStore } from "react";

import { useListTransfersSuspense } from "../../../../api/gen/query/transfers/transfers";

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
      <Suspense fallback={null}>
        <MovesOnLink controllerId={landing.nodes[0]?.id ?? ""} worker={worker} />
      </Suspense>
      {link.dialer !== undefined && (
        <Details summary={`${dialer} connected to ${listener}.`}>
          <p>The moving dots are brightest at the Node that opened the connection.</p>
        </Details>
      )}
    </>
  );
}

/** The Moves whose files cross the link now, each with the way it goes. */
function MovesOnLink({ controllerId, worker }: { controllerId: string; worker: NodeRow }) {
  const { transfers } = useListTransfersSuspense({ nodeId: worker.id }).data;
  const moving = transfers.filter(
    ({ purpose, state, source, destination }) =>
      purpose === "move" &&
      state !== "failed" &&
      (source.nodeId === controllerId || destination.nodeId === controllerId),
  );
  if (moving.length === 0) {
    return null;
  }
  return (
    <InspectorSection title="Moving on this link">
      <ul className="flex flex-col gap-1">
        {moving.map((transfer) => (
          <li key={transfer.id}>
            {transfer.jobName}, from {transfer.source.nodeName} to {transfer.destination.nodeName}
          </li>
        ))}
      </ul>
    </InspectorSection>
  );
}
