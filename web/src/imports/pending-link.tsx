import { ArrowRight01Icon } from "@hugeicons/core-free-icons";
import { Link } from "@tanstack/react-router";

import { useCounts } from "../shell/counts";
import { Icon } from "../ui/icon";
import { List, ListRow } from "../ui/list";
import { needYouWords } from "./words";

/** The Library's row to Pending imports while folders there need the Admin. */
export function PendingLink() {
  const count = useCounts()?.pendingImports ?? 0;
  if (count === 0) {
    return null;
  }
  return (
    <div className="mb-4">
      <List label="Pending imports">
        <ListRow
          title={needYouWords(count)}
          meta="A library scan found them and can't match them with confidence."
          trailing={<Icon icon={ArrowRight01Icon} className="size-4" />}
          render={<Link to="/library/pending" />}
        />
      </List>
    </div>
  );
}
