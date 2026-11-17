import { useQueryClient, useSuspenseInfiniteQuery } from "@tanstack/react-query";
import { useEffect, useRef } from "react";

import type { NotificationsFeedRow as Row } from "../api/gen/model";
import {
  markNotificationsSeen,
  useGetNotificationSettingsSuspense,
} from "../api/gen/query/notifications/notifications";
import { type Clock, durationWords, sinceWords } from "../format";
import { adoptSeen } from "../live/adopt-notifications";
import { useNow } from "../tasks/use-now";
import { Button } from "../ui/button";
import { feedQuery } from "./feed-keys";
import { settingKeys, settingOf } from "./settings-panel";

const minute = 60_000;

function FeedRow({ row, clock }: { row: Row; clock: Clock }) {
  const body = (
    <>
      <span className="flex items-baseline gap-2">
        {row.unread === true && (
          <span className="size-1.5 shrink-0 self-center rounded-capsule bg-glow">
            <span className="sr-only">Unread</span>
          </span>
        )}
        <span className="min-w-0 flex-1 font-medium text-ink-1">{row.title}</span>
        <time
          dateTime={row.at}
          title={new Date(row.at).toLocaleString()}
          className="shrink-0 text-meta text-ink-3"
        >
          {sinceWords(row.at, clock)}
        </time>
      </span>
      {row.line !== "" && <span className="text-meta text-ink-2">{row.line}</span>}
    </>
  );
  const look = "flex flex-col gap-0.5 rounded-item px-3 py-2";
  return (
    <li>
      {row.link === null ? (
        <div className={look}>{body}</div>
      ) : (
        <a href={row.link} className={`${look} hover:bg-fill`}>
          {body}
        </a>
      )}
    </li>
  );
}

/** Marks every row the first page shows as read, once, after it renders. */
function useSeen(newest: number | undefined) {
  const queryClient = useQueryClient();
  const posted = useRef(false);
  useEffect(() => {
    if (newest === undefined || posted.current) {
      return;
    }
    posted.current = true;
    const seen = async () => {
      await markNotificationsSeen({ through: newest });
      adoptSeen(queryClient);
    };
    void seen();
  }, [newest, queryClient]);
}

function EmptyFeed() {
  const { settings } = useGetNotificationSettingsSuspense().data;
  const keep = settingOf(settings, settingKeys.keepFor);
  return (
    <p className="text-ink-2">
      No notifications in the last {keep.kind === "duration" ? durationWords(keep.value) : ""}.
    </p>
  );
}

/** What the bell showed, newest first. Opening it marks what it shows as read. */
export function FeedSheet() {
  const { data, hasNextPage, fetchNextPage, isFetchingNextPage } =
    useSuspenseInfiniteQuery(feedQuery);
  const clock = { now: useNow(minute), zone: undefined };
  const rows = data.pages.flatMap((page) => page.rows);
  useSeen(rows[0]?.seq);
  if (rows.length === 0) {
    return <EmptyFeed />;
  }
  return (
    <div className="flex flex-col gap-3">
      <ul aria-label="Notifications" className="-mx-3 flex flex-col">
        {rows.map((row) => (
          <FeedRow key={row.seq} row={row} clock={clock} />
        ))}
      </ul>
      {hasNextPage && (
        <div>
          <Button
            busy={isFetchingNextPage}
            onClick={() => {
              void fetchNextPage();
            }}
          >
            Show older notifications
          </Button>
        </div>
      )}
    </div>
  );
}
