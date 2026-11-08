import type { TrackersTrackerItem } from "../api/gen/model";
import { useListDownloadClientsSuspense } from "../api/gen/query/downloadclients/downloadclients";
import { useListNodesSuspense } from "../api/gen/query/nodes/nodes";
import { useListTrackerRoutesSuspense } from "../api/gen/query/trackers/trackers";
import { clientsOf } from "../grabs/send-to-sheet";
import { Empty } from "../shell/empty";
import { RouteRow } from "./route-row";
import { announcesWords, clientNames, unroutedWords } from "./route-words";

const contentLook = "max-w-[640px]";

function Routes({ tracker }: { tracker: TrackersTrackerItem }) {
  const { routes } = useListTrackerRoutesSuspense(tracker.id).data;
  const list = useListDownloadClientsSuspense().data;
  const names = clientNames(useListNodesSuspense().data.nodes, list);
  const picks = clientsOf(list.cards);
  if (routes.length === 0) {
    return <Empty>No Indexer reaches {tracker.name}, so it has no routes.</Empty>;
  }
  return (
    <div className={contentLook}>
      {routes.map((route) => (
        <RouteRow
          key={route.homeNodeId}
          tracker={tracker}
          route={route}
          picks={picks}
          names={names}
          announces={announcesWords(list.cards, names, route.to)}
        />
      ))}
    </div>
  );
}

/** Where a private Tracker's Grabs download, per home Node of its Indexers. */
export function RoutingTab({ tracker }: { tracker: TrackersTrackerItem }) {
  const { privacy } = tracker;
  return privacy === "private" ? (
    <Routes tracker={tracker} />
  ) : (
    <p className={`${contentLook} text-ink-2`}>{unroutedWords(tracker.name, privacy)}</p>
  );
}
