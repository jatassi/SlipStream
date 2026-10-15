import { Logout01Icon } from "@hugeicons/core-free-icons";
import { useNavigate } from "@tanstack/react-router";

import { useAccount } from "../../session/session";
import { Count } from "../../ui/count";
import { Sheet, SheetCancel, SheetGroup, SheetItem, SheetToggle } from "../../ui/sheet";
import { Avatar, startupLabel, useSignOut, useStartupOn } from "../account";
import { countDots, useCounts } from "../counts";
import type { ShownRow } from "../nav";
import { setMoreOpen, useMoreOpen } from "./more-state";
import { Version } from "./version";

export function MoreSheet({ rows }: { rows: readonly ShownRow[] }) {
  const navigate = useNavigate();
  const counts = useCounts();
  const account = useAccount();
  const signOut = useSignOut();
  const [startupOn, setStartupOn] = useStartupOn();
  return (
    <Sheet label="More" open={useMoreOpen()} onOpenChange={setMoreOpen}>
      <SheetGroup>
        {rows.map((row) => {
          const count = row.count === undefined ? 0 : (counts?.[row.count] ?? 0);
          return (
            <SheetItem
              key={row.id}
              icon={row.icon}
              trailing={
                row.count !== undefined && count > 0 ? (
                  <Count value={count} dot={countDots[row.count]} />
                ) : undefined
              }
              onClick={() => {
                void navigate({ to: row.to });
              }}
            >
              {row.label}
            </SheetItem>
          );
        })}
      </SheetGroup>
      <SheetGroup>
        <div className="flex h-(--menu-item-height) items-center gap-3 px-4">
          <Avatar name={account.displayName} />
          <span className="min-w-0 flex-1 truncate text-body-touch text-ink-1">
            {account.displayName}
          </span>
        </div>
        <SheetItem icon={Logout01Icon} onClick={signOut}>
          Sign out
        </SheetItem>
        <SheetToggle checked={startupOn} onCheckedChange={setStartupOn}>
          {startupLabel}
        </SheetToggle>
      </SheetGroup>
      <Version />
      <SheetCancel />
    </Sheet>
  );
}
