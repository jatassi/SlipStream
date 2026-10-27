import type { IndexersIndexer, IndexersQuery, IndexersTracker, NodesNode } from "../api/gen/model";
import { useShell } from "../ui/shell-context";
import { IndexerActions } from "./indexer-actions";
import { IndexerFacts } from "./indexer-facts";
import { IndexerMachinery, RecentQueries } from "./indexer-history";
import { StatusLine } from "./status-line";
import { IndexerSwitch } from "./switches";

/** Waiting on a lapsed home Node, where Allow failover lets it search from another. */
const waitsOnLapsedHome = (indexer: IndexersIndexer) =>
  indexer.signsIn && indexer.waits?.kind === "waiting" && indexer.waits.lapsed;

function Lead({ indexer }: { indexer: IndexersIndexer }) {
  const type = useShell() === "phone" ? "text-sentence-touch" : "text-sentence";
  return (
    <div className="flex flex-col gap-2">
      <div className={type}>
        <StatusLine indexer={indexer} />
      </div>
      {indexer.fix !== null && <p className="text-ink-1">{indexer.fix}</p>}
      {waitsOnLapsedHome(indexer) && (
        <div id="allow-failover" className="scroll-mt-20 self-start">
          <IndexerSwitch indexer={indexer} name="allowFailover" />
        </div>
      )}
    </div>
  );
}

/** An Indexer's page: its status and what to do first, then its facts, then what it has been doing. */
export function IndexerPage({
  indexer,
  queries,
  nodes,
  trackers,
}: {
  indexer: IndexersIndexer;
  queries: readonly IndexersQuery[];
  nodes: readonly NodesNode[];
  trackers: readonly IndexersTracker[];
}) {
  return (
    <div className="flex max-w-[860px] flex-col gap-6">
      <Lead indexer={indexer} />
      <IndexerActions indexer={indexer} nodes={nodes} />
      <IndexerFacts indexer={indexer} nodes={nodes} trackers={trackers} />
      <RecentQueries queries={queries} />
      <IndexerMachinery indexer={indexer} nodes={nodes} />
    </div>
  );
}
