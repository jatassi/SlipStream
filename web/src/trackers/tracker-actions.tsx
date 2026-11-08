import { useNavigate } from "@tanstack/react-router";

import type { TrackersTrackerItem } from "../api/gen/model";
import { TrackerMenu } from "../indexers/tracker-menu";
import { useTrackers } from "./tracker";

/** The Tracker page's menu; once the Tracker is gone it leaves for the kept Tracker or the list. */
export function TrackerActions({ tracker }: { tracker: TrackersTrackerItem }) {
  const navigate = useNavigate();
  const others = useTrackers().filter((other) => other.id !== tracker.id);
  return (
    <TrackerMenu
      tracker={tracker}
      others={others}
      onGone={(kept) =>
        kept === undefined
          ? navigate({ to: "/settings/$section", params: { section: "trackers" } })
          : navigate({ to: "/settings/trackers/$trackerId", params: { trackerId: kept.id } })
      }
    />
  );
}
