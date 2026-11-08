import { Link } from "@tanstack/react-router";

import type { TrackersTrackerItem } from "../api/gen/model";
import { Chevron } from "../ui/chevron";
import { Count } from "../ui/count";
import { Icon } from "../ui/icon";
import { List, ListRow } from "../ui/list";
import { type TrackerSection, trackerSections } from "./sections";

export function SectionCount({
  section,
  tracker,
}: {
  section: TrackerSection;
  tracker: TrackersTrackerItem;
}) {
  const count = section.count?.(tracker) ?? 0;
  const dot = section.dot?.(tracker);
  return count > 0 || dot !== undefined ? <Count value={count} {...(dot && { dot })} /> : null;
}

/** On phone, the Tracker's top page lists its other sections, each a pushed page. */
export function SectionRows({ tracker }: { tracker: TrackersTrackerItem }) {
  return (
    <div className="mb-7">
      <List label={tracker.name}>
        {trackerSections
          .filter((section) => section.id !== "torrents")
          .map((section) => (
            <ListRow
              key={section.id}
              title={section.label}
              leading={<Icon icon={section.icon} className="size-5 shrink-0 text-ink-2" />}
              trailing={
                <span className="flex items-center gap-2">
                  <SectionCount section={section} tracker={tracker} />
                  <Chevron />
                </span>
              }
              render={<Link to={section.to} params={{ trackerId: tracker.id }} />}
            />
          ))}
      </List>
    </div>
  );
}
