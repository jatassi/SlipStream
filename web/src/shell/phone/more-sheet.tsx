import { Logout01Icon } from "@hugeicons/core-free-icons";
import { useNavigate } from "@tanstack/react-router";

import { useAccount } from "../../session/session";
import { Sheet, SheetCancel, SheetGroup, SheetItem, SheetToggle } from "../../ui/sheet";
import { Avatar, startupLabel, toPortal, useSignOut, useStartupOn } from "../account";
import type { ShownRow } from "../nav";
import { MoreNav } from "./more-nav";
import { setMoreOpen, useMoreOpen } from "./more-state";
import { Version } from "./version";

export function MoreSheet({ rows }: { rows: readonly ShownRow[] }) {
  const navigate = useNavigate();
  const account = useAccount();
  const signOut = useSignOut();
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
          icon={toPortal.icon}
          onClick={() => {
            void navigate({ to: toPortal.to });
          }}
        >
          {toPortal.label}
        </SheetItem>
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
