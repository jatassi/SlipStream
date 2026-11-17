import { useQuery } from "@tanstack/react-query";

import { listNotifications } from "../api/gen/query/notifications/notifications";
import { BellFeed } from "./bell-feed";
import { unreadKey, unreadParams } from "./feed-keys";

const countCap = 100;

const countWords = (unread: number) => (unread >= countCap ? "99+" : String(unread));

/** The bell: its unread count, opening what it showed. */
export function Bell() {
  const unread =
    useQuery({
      queryKey: unreadKey,
      queryFn: ({ signal }) => listNotifications(unreadParams, { signal }),
    }).data?.unread ?? 0;
  return (
    <span className="relative inline-flex">
      <BellFeed
        label={unread === 0 ? "Notifications" : `Notifications, ${countWords(unread)} unread`}
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
  );
}
