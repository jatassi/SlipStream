import { useNavigate } from "@tanstack/react-router";
import { Fragment } from "react";

import { Count } from "../../ui/count";
import { SheetGroup, SheetItem } from "../../ui/sheet";
import { type CountKey, countDots, useCounts } from "../counts";
import type { ShownRow } from "../nav";

/** Every nav row the tab bar doesn't hold, and every row's children, in nav order. */
export function MoreNav({ rows }: { rows: readonly ShownRow[] }) {
  const navigate = useNavigate();
  return (
    <SheetGroup>
      {rows.map((row) => (
        <Fragment key={row.id}>
          {row.tab === undefined && (
            <SheetItem
              icon={row.icon}
              trailing={row.count === undefined ? undefined : <RowCount count={row.count} />}
              onClick={() => {
                void navigate({ to: row.to });
              }}
            >
              {row.label}
            </SheetItem>
          )}
          {row.children.map((child) => (
            <SheetItem
              key={child.id}
              icon={child.icon}
              onClick={() => {
                void navigate({ to: child.to, params: child.params });
              }}
            >
              {child.label}
            </SheetItem>
          ))}
        </Fragment>
      ))}
    </SheetGroup>
  );
}

function RowCount({ count }: { count: CountKey }) {
  const value = useCounts()?.[count] ?? 0;
  return value > 0 ? <Count value={value} dot={countDots[count]} /> : null;
}
