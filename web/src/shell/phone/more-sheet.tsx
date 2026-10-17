import { UserCircleIcon } from "@hugeicons/core-free-icons";
import { useNavigate } from "@tanstack/react-router";

import { useAccount } from "../../session/session";
import { Sheet, SheetCancel, SheetGroup, SheetItem, SheetToggle } from "../../ui/sheet";
import { Avatar, startupLabel, toPortal, useLeaveActions, useStartupOn } from "../account";
import type { ShownRow } from "../nav";
import { MoreNav } from "./more-nav";
import { setMoreOpen, useMoreOpen } from "./more-state";
import { Version } from "./version";

export function MoreSheet({ rows }: { rows: readonly ShownRow[] }) {
  const navigate = useNavigate();
  const account = useAccount();
  const [startupOn, setStartupOn] = useStartupOn();
  return (
    <Sheet label="More" open={useMoreOpen()} onOpenChange={setMoreOpen}>
      <MoreNav rows={rows} />
      <SheetGroup>
        <div className="flex h-(--menu-item-height) items-center gap-3 px-4">
          <Avatar name={account.displayName} />
          <span className="min-w-0 flex-1 truncate text-sentence-touch text-ink-1">
            {account.displayName}
          </span>
        </div>
        <SheetItem
          icon={UserCircleIcon}
          onClick={() => {
            void navigate({ to: "/account" });
          }}
        >
          Your Account
        </SheetItem>
        <SheetItem
          icon={toPortal.icon}
          onClick={() => {
            void navigate({ to: toPortal.to });
          }}
        >
          {toPortal.label}
        </SheetItem>
        {useLeaveActions().map((action) => (
          <SheetItem key={action.label} icon={action.icon} onClick={action.onSelect}>
            {action.label}
          </SheetItem>
        ))}
        <SheetToggle checked={startupOn} onCheckedChange={setStartupOn}>
          {startupLabel}
        </SheetToggle>
      </SheetGroup>
      <Version />
      <SheetCancel />
    </Sheet>
  );
}
