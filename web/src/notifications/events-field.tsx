import type {
  NotificationsEvent as Event,
  NotificationsEventGroup as Group,
} from "../api/gen/model";
import { ChoiceSet } from "../indexers/add/families-field";

const groupLabels: { readonly [G in Group]: string } = {
  grabs: "Grabs",
  imports: "Imports",
  library: "Library",
  subtitles: "Subtitles",
  health: "Health",
  backups: "Backups",
  moves: "Moves",
  system: "System",
};

/** Every event a connection or the bell can get, one set of checkboxes per group. A fixed event, such as Test, isn't offered. */
export function EventsChoice({
  events,
  chosen,
  errors,
  onChange,
}: {
  events: readonly Event[];
  chosen: readonly string[];
  errors: readonly unknown[];
  onChange: (chosen: string[]) => void;
}) {
  const offered = events.filter((event) => !event.fixed);
  const groups = [...new Set(offered.map((event) => event.group))];
  return (
    <div className="flex flex-col gap-3">
      {groups.map((group, at) => (
        <ChoiceSet
          key={group}
          legend={groupLabels[group]}
          options={offered
            .filter((event) => event.group === group)
            .map((event) => ({ value: event.id, label: event.label }))}
          chosen={chosen}
          errors={at === groups.length - 1 ? errors : []}
          onChange={onChange}
        />
      ))}
    </div>
  );
}
