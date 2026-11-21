import { BellIcon } from "@hugeicons/core-free-icons";
import { Suspense, useState } from "react";

import { Dialog } from "../ui/dialog";
import { IconButton } from "../ui/icon-button";
import { Popover } from "../ui/popover";
import { useShell } from "../ui/shell-context";
import { Skeleton } from "../ui/skeleton";
import { FeedSheet } from "./feed-sheet";

function Feed({ onFollow }: { onFollow: () => void }) {
  return (
    <Suspense fallback={<Skeleton label="Loading notifications" shape="rows" lines={2} />}>
      <FeedSheet onFollow={onFollow} />
    </Suspense>
  );
}

/** The bell's button and what it opens: a Popover on the bell, a bottom sheet on phone. */
export function BellFeed({ label }: { label: string }) {
  const [open, setOpen] = useState(false);
  const close = () => {
    setOpen(false);
  };
  if (useShell() === "phone") {
    return (
      <>
        <IconButton
          icon={BellIcon}
          label={label}
          onClick={() => {
            setOpen(true);
          }}
        />
        <Dialog title="Notifications" open={open} onOpenChange={setOpen}>
          {open && <Feed onFollow={close} />}
        </Dialog>
      </>
    );
  }
  return (
    <Popover
      label="Notifications"
      width={400}
      open={open}
      onOpenChange={setOpen}
      trigger={<IconButton icon={BellIcon} label={label} />}
    >
      <div className="flex max-h-[min(560px,calc(var(--available-height)-32px))] flex-col gap-3">
        <h2 className="text-heading">Notifications</h2>
        <div className="-mx-4 min-h-0 overflow-y-auto overscroll-contain px-4">
          <Feed onFollow={close} />
        </div>
      </div>
    </Popover>
  );
}
