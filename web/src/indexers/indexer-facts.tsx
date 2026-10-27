import type { ReactNode } from "react";

import type {
  IndexersIndexer,
  IndexersLimitUse,
  IndexersTracker,
  NodesNode,
} from "../api/gen/model";
import { KeyValues } from "../routes/_app/nodes/-facts";
import { Meter } from "../ui/meter";
import { Section } from "../ui/section";
import { SettingGroup } from "../ui/setting-field";
import { IndexerSwitch, type SwitchName } from "./switches";
import { kindWords, limitWords, nodeName } from "./words";

function LimitUse({ label, limit }: { label: string; limit: IndexersLimitUse }) {
  return (
    <span className="flex flex-col gap-1.5">
      <span>{limitWords(limit)}</span>
      {limit.count > 0 && (
        <Meter
          label={`${label} used`}
          value={limit.used}
          max={limit.count}
          tone={limit.used >= limit.count ? "warn" : undefined}
        />
      )}
    </span>
  );
}

function trackerWords(indexer: IndexersIndexer, trackers: readonly IndexersTracker[]): string {
  const names = trackers
    .filter((tracker) => indexer.trackerIds.includes(tracker.id))
    .map((tracker) => tracker.name);
  return names.length === 0 ? "No Tracker yet" : names.join(", ");
}

const words = (text: string): ReactNode => <span>{text}</span>;

/** What the Indexer reaches, where it searches from, its limits and its switches. */
export function IndexerFacts({
  indexer,
  nodes,
  trackers,
}: {
  indexer: IndexersIndexer;
  nodes: readonly NodesNode[];
  trackers: readonly IndexersTracker[];
}) {
  const switches: SwitchName[] = indexer.signsIn
    ? ["on", "automaticSearch", "rss", "allowFailover"]
    : ["on", "automaticSearch", "rss"];
  return (
    <Section title="Searching">
      <div className="flex flex-col gap-3">
        <KeyValues
          rows={[
            ["Tracker", words(trackerWords(indexer, trackers))],
            ["Searches from", words(nodeName(nodes, indexer.homeNodeId))],
            ["Kind", words(kindWords[indexer.kind])],
            ["Priority", words(String(indexer.priority))],
            ["Searches", <LimitUse key="searches" label="Searches" limit={indexer.searches} />],
            ["Grabs", <LimitUse key="grabs" label="Grabs" limit={indexer.grabs} />],
          ]}
        />
        <SettingGroup>
          <div className="flex flex-wrap gap-x-6">
            {switches.map((name) => (
              <IndexerSwitch key={name} indexer={indexer} name={name} />
            ))}
          </div>
        </SettingGroup>
      </div>
    </Section>
  );
}
