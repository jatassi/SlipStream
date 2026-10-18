import { useNavigate } from "@tanstack/react-router";
import { Fragment } from "react";

import { AlertDot, Count } from "../../ui/count";
import { SheetGroup, SheetItem } from "../../ui/sheet";
import { type AlertKey, type CountKey, alertOf, countDots, useCounts } from "../counts";
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
              trailing={<RowTrailing count={row.count} alert={row.alert} />}
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

function RowTrailing({
  count,
  alert,
}: {
  count: CountKey | undefined;
  alert: AlertKey | undefined;
}) {
  const counts = useCounts();
  const value = count === undefined ? 0 : (counts?.[count] ?? 0);
  const tone = alertOf(counts, alert);
  return (
    <>
      {count !== undefined && value > 0 && <Count value={value} dot={countDots[count]} />}
      {tone !== undefined && <AlertDot tone={tone} />}
    </>
  );
}
