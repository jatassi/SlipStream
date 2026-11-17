import { ArrowRight01Icon } from "@hugeicons/core-free-icons";
import { Link } from "@tanstack/react-router";

import type { WorksContentType } from "../api/gen/model";
import { Icon } from "../ui/icon";
import { List, ListRow } from "../ui/list";

/**
 * The Library's way to its Calendar, always shown: the Calendar of the one
 * Content Type in view, or of all. A Library, Wanted and Calendar switch
 * would replace this one component.
 */
export function CalendarEntry({ types }: { types: readonly WorksContentType[] }) {
  const [only] = types;
  return (
    <div className="mb-4">
      <List label="Calendar">
        <ListRow
          title="Calendar"
          meta="Release dates in your library, day by day."
          trailing={<Icon icon={ArrowRight01Icon} className="size-4" />}
          render={
            types.length === 1 && only !== undefined ? (
              <Link to="/library/calendar/$contentType" params={{ contentType: only.id }} />
            ) : (
              <Link to="/library/calendar" />
            )
          }
        />
      </List>
    </div>
  );
}
