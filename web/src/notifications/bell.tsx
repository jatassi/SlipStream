import { BellIcon } from "@hugeicons/core-free-icons";
import { Suspense, useState } from "react";

import { useQuery } from "@tanstack/react-query";

import { listNotifications } from "../api/gen/query/notifications/notifications";
import { Dialog } from "../ui/dialog";
import { IconButton } from "../ui/icon-button";
import { Skeleton } from "../ui/skeleton";
import { unreadKey, unreadParams } from "./feed-keys";
import { FeedSheet } from "./feed-sheet";

const countCap = 100;

const countWords = (unread: number) => (unread >= countCap ? "99+" : String(unread));

/** The bell: its unread count, opening what it showed. A Dialog, a bottom sheet on phone. */
export function Bell() {
  const unread =
    useQuery({
      queryKey: unreadKey,
      queryFn: ({ signal }) => listNotifications(unreadParams, { signal }),
    }).data?.unread ?? 0;
  const [open, setOpen] = useState(false);
  return (
    <>
      <span className="relative inline-flex">
        <IconButton
          icon={BellIcon}
          label={unread === 0 ? "Notifications" : `Notifications, ${countWords(unread)} unread`}
          onClick={() => {
            setOpen(true);
          }}
        />
        {unread > 0 && (
          <span
            aria-hidden="true"
            className="pointer-events-none absolute -top-0.5 -right-0.5 grid h-4 min-w-4 place-items-center rounded-capsule bg-solid px-1 text-[10px] font-semibold text-on-solid tabular-nums"
          >
            {countWords(unread)}
          </span>
        )}
      </span>
      <Dialog title="Notifications" open={open} onOpenChange={setOpen}>
        {open && (
          <Suspense fallback={<Skeleton label="Loading notifications" />}>
            <FeedSheet />
          </Suspense>
        )}
      </Dialog>
    </>
  );
}
