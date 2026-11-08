import { Link } from "@tanstack/react-router";

import type { DownloadclientsRoutedTracker } from "../api/gen/model";

/** The private Trackers whose Grabs download on this client now, each a link to its Routing tab. */
export function RoutedHere({ trackers }: { trackers: readonly DownloadclientsRoutedTracker[] }) {
  if (trackers.length === 0) {
    return null;
  }
  return (
    <div className="flex flex-wrap items-baseline gap-x-2 gap-y-1 text-meta text-ink-3 in-data-[density=touch]:text-body-touch">
      <span>Trackers routed here</span>
      <ul aria-label="Trackers routed here" className="flex flex-wrap gap-x-3 gap-y-1">
        {trackers.map((tracker) => (
          <li key={tracker.id}>
            <Link
              to="/settings/trackers/$trackerId/routing"
              params={{ trackerId: tracker.id }}
              className="text-ink-2 underline hover:text-ink-1"
            >
              {tracker.name}
            </Link>
            {!tracker.stored && " (default)"}
          </li>
        ))}
      </ul>
    </div>
  );
}
