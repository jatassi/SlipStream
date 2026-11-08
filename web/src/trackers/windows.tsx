import { useSuspenseQueries } from "@tanstack/react-query";
import { Link } from "@tanstack/react-router";
import { startTransition, useState } from "react";

import type { ListTrackerWindowsState, TrackersWindow } from "../api/gen/model";
import {
  getListTrackerWindowsQueryKey,
  getListTrackerWindowsSuspenseQueryOptions,
  listTrackerWindows,
} from "../api/gen/query/trackers/trackers";
import type { Clock } from "../format";
import { useNow } from "../tasks/use-now";
import { useLanes } from "./window-lanes";
import { WindowList } from "./window-list";

type State = ListTrackerWindowsState | undefined;

const paramsOf = (state: State, offset: number) =>
  state === undefined && offset === 0
    ? undefined
    : { ...(state && { state }), ...(offset > 0 && { offset }) };

/** The open windows' query and the whole history's, each from its first page. */
export const firstWindowPages = (trackerId: string) =>
  (["owing", undefined] as const).map((state) =>
    getListTrackerWindowsSuspenseQueryOptions(trackerId, paramsOf(state, 0)),
  );

function usePages(trackerId: string, state: State) {
  const [offsets, setOffsets] = useState<readonly number[]>([0]);
  const pages = useSuspenseQueries({
    queries: offsets.map((offset) => {
      const params = paramsOf(state, offset);
      return {
        queryKey: getListTrackerWindowsQueryKey(trackerId, params),
        queryFn: ({ signal }: { signal: AbortSignal }) =>
          listTrackerWindows(trackerId, params, { signal }),
      };
    }),
  }).map((page) => page.data);
  const next = pages.at(-1)?.nextOffset ?? null;
  return {
    windows: pages.flatMap((page) => page.windows),
    more:
      next === null
        ? undefined
        : () => {
            startTransition(() => {
              setOffsets((shown) => [...shown, next]);
            });
          },
  };
}

const wordsTick = 60_000;

/** A Tracker's open windows with what each still owes, then its met, missed and released ones. */
export function Windows({ trackerId, name }: { trackerId: string; name: string }) {
  const clock: Clock = { now: useNow(wordsTick), zone: undefined };
  const open = usePages(trackerId, "owing");
  const all = usePages(trackerId, undefined);
  const lanes = useLanes();
  const link = (window: TrackersWindow) => {
    const [torrentId] = window.torrents;
    return torrentId === undefined ? undefined : (
      <Link
        to="/settings/trackers/$trackerId/torrents/$torrentId"
        params={{ trackerId, torrentId }}
      />
    );
  };
  return (
    <>
      <WindowList
        title="Open windows"
        windows={open.windows}
        more={open.more}
        empty={`No torrent owes ${name} seeding.`}
        tracker={name}
        clock={clock}
        lanes={lanes}
        link={link}
      />
      <WindowList
        title="History"
        windows={all.windows.filter((window) => window.state !== "owing")}
        more={all.more}
        empty="No hit and run window has ended yet."
        tracker={name}
        clock={clock}
        link={link}
      />
    </>
  );
}
