import type { GrabsGrab } from "../api/gen/model";
import { useListIndexersSuspense } from "../api/gen/query/indexers/indexers";
import { useListNodesSuspense } from "../api/gen/query/nodes/nodes";
import { useListTargetsSuspense } from "../api/gen/query/targets/targets";
import { useListTrackersSuspense } from "../api/gen/query/trackers/trackers";
import { bytes } from "../format";
import { useQualityNames } from "../targets/quality-names";
import { type Facts, FactList } from "../torrents/inspector-facts";
import { Data } from "../ui/data";
import { Details } from "../ui/details";

function useNamesOf(grab: GrabsGrab) {
  const { nodes } = useListNodesSuspense().data;
  const { indexers } = useListIndexersSuspense().data;
  const { trackers } = useListTrackersSuspense().data;
  const { targets } = useListTargetsSuspense().data;
  const quality = useQualityNames();
  return {
    node: (id: string) => nodes.find((node) => node.id === id)?.name ?? "A removed Node",
    indexer: indexers.find(({ id }) => id === grab.indexerId)?.name ?? "A deleted Indexer",
    tracker: trackers.find(({ id }) => id === grab.trackerId)?.name,
    targets: grab.targetIds
      .map((id) => targets.find((target) => target.id === id)?.name ?? "A deleted Target")
      .join(", "),
    quality: grab.quality === undefined ? undefined : quality(grab.quality),
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
  const facts: Facts = [
    [
      "Release",
      <span key="r" className="flex flex-col">
        {grab.releaseWords}
        <Data className="break-all text-ink-3">{grab.releaseTitle}</Data>
      </span>,
    ],
    ...(names.quality === undefined ? [] : [["Quality", names.quality] as const]),
    ["For", names.targets],
    ["Indexer", names.indexer],
    ...(names.tracker === undefined ? [] : [["Tracker", names.tracker] as const]),
    ["Client", clientWords(grab, names.node)],
  ];
  return (
    <>
      <FactList facts={facts} />
      <Details>
        <FactList
          facts={[
            ["Size", bytes(grab.releaseSize)],
            [
              "Info hash",
              grab.infoHash === null ? "Not fetched yet" : <Data key="h">{grab.infoHash}</Data>,
            ],
          ]}
        />
      </Details>
    </>
  );
}
