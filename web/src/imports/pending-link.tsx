import { ArrowRight01Icon } from "@hugeicons/core-free-icons";
import { Link } from "@tanstack/react-router";

import { useCounts } from "../shell/counts";
import { Icon } from "../ui/icon";
import { List, ListRow } from "../ui/list";
import { pendingWords } from "./words";

/**
 * The Library's row to Pending imports while an Import there has a file
 * waiting for the Admin. `preload={false}` keeps a hover from reading the
 * Library, which that page's loader does.
 */
export function PendingLink(link: { preload?: false }) {
  const count = useCounts()?.pendingImports ?? 0;
  if (count === 0) {
    return null;
  }
  return (
    <div className="mb-4">
      <List label="Pending imports">
        <ListRow
          title={pendingWords(count)}
          meta="Each has a file SlipStream can't import without you."
          trailing={<Icon icon={ArrowRight01Icon} className="size-4" />}
          render={<Link to="/library/pending" {...link} />}
        />
      </List>
    </div>
  );
}
