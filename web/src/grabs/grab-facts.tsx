import type { GrabsGrab } from "../api/gen/model";
import { useListIndexersSuspense } from "../api/gen/query/indexers/indexers";
import { useListNodesSuspense } from "../api/gen/query/nodes/nodes";
import { useListTargetsSuspense } from "../api/gen/query/targets/targets";
import { useListTrackersSuspense } from "../api/gen/query/trackers/trackers";
import { ReleaseName } from "../release/row";
import { Data } from "../ui/data";
import { Details } from "../ui/details";
import { Fact, Facts } from "../ui/facts";

function useNamesOf(grab: GrabsGrab) {
  const { nodes } = useListNodesSuspense().data;
  const { indexers } = useListIndexersSuspense().data;
  const { trackers } = useListTrackersSuspense().data;
  const { targets } = useListTargetsSuspense().data;
  return {
    node: (id: string) => nodes.find((node) => node.id === id)?.name ?? "A removed Node",
    indexer: indexers.find(({ id }) => id === grab.indexerId)?.name ?? "A deleted Indexer",
    tracker: trackers.find(({ id }) => id === grab.trackerId)?.name,
    targets: grab.targetIds
      .map((id) => targets.find((target) => target.id === id)?.name ?? "A deleted Target")
      .join(", "),
  };
}

function clientWords(grab: GrabsGrab, node: (id: string) => string): string {
  if (grab.waitsFor !== undefined) {
    const name = grab.waitsFor.client?.name ?? "Built-in client";
    return `${name} on ${node(grab.waitsFor.nodeId)}, which is offline`;
  }
  if (grab.torrentId === null) {
    return "Chosen once SlipStream fetches the Release";
  }
  return `${grab.downloadClient?.name ?? "Built-in client"} on ${node(grab.nodeId)}`;
}

export function GrabFacts({ grab }: { grab: GrabsGrab }) {
  const names = useNamesOf(grab);
  return (
    <>
      <Facts look="inline">
        <Fact label="Release">
          <ReleaseName
            release={{ title: grab.releaseTitle, facts: grab.releaseFacts, size: grab.releaseSize }}
          />
        </Fact>
        <Fact label="For">{names.targets}</Fact>
        <Fact label="Indexer">{names.indexer}</Fact>
        {names.tracker !== undefined && <Fact label="Tracker">{names.tracker}</Fact>}
        <Fact label="Client">{clientWords(grab, names.node)}</Fact>
      </Facts>
      <Details>
        <Facts look="well">
          <Fact label="Info hash">
            {grab.infoHash === null ? "Not fetched yet" : <Data>{grab.infoHash}</Data>}
          </Fact>
        </Facts>
      </Details>
    </>
  );
}
