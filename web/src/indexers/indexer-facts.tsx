import type {
  IndexersIndexer,
  IndexersLimitUse,
  TrackersTracker,
  NodesNode,
} from "../api/gen/model";
import { Data } from "../ui/data";
import { Fact, Facts } from "../ui/facts";
import { Meter } from "../ui/meter";
import { Section } from "../ui/section";
import { SettingGroup } from "../ui/setting-field";
import { RSSFetch } from "./rss-fetch";
import { IndexerSwitch, type SwitchName } from "./switches";
import { kindOf, limitWords, nodeName } from "./words";

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

function trackerWords(indexer: IndexersIndexer, trackers: readonly TrackersTracker[]): string {
  const names = trackers
    .filter((tracker) => indexer.trackerIds.includes(tracker.id))
    .map((tracker) => tracker.name);
  return names.length === 0 ? "No Tracker yet" : names.join(", ");
}

/** What the Indexer reaches, where it searches from, its limits and its switches. */
export function IndexerFacts({
  indexer,
  nodes,
  trackers,
}: {
  indexer: IndexersIndexer;
  nodes: readonly NodesNode[];
  trackers: readonly TrackersTracker[];
}) {
  const username = (indexer.definition ?? indexer.port)?.values["username"];
  const switches: SwitchName[] = indexer.signsIn
    ? ["on", "automaticSearch", "rss", "allowFailover"]
    : ["on", "automaticSearch", "rss"];
  return (
    <Section title="Searching">
      <div className="flex flex-col gap-3">
        <Facts look="sheet">
          <Fact label="Tracker">{trackerWords(indexer, trackers)}</Fact>
          {username !== undefined && (
            <Fact label="Username">
              <Data>{username}</Data>
            </Fact>
          )}
          <Fact label="Searches from">{nodeName(nodes, indexer.homeNodeId)}</Fact>
          <Fact label="Kind">{kindOf(indexer)}</Fact>
          <Fact label="Priority">{indexer.priority}</Fact>
          <Fact label="Searches">
            <LimitUse label="Searches" limit={indexer.searches} />
          </Fact>
          <Fact label="Grabs">
            <LimitUse label="Grabs" limit={indexer.grabs} />
          </Fact>
          {indexer.rss && (
            <Fact label="RSS sync">
              <RSSFetch indexer={indexer} />
            </Fact>
          )}
        </Facts>
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
