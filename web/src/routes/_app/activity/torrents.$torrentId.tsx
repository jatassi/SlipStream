import { createFileRoute, notFound, useNavigate } from "@tanstack/react-router";
import { useEffect } from "react";

import {
  getListTorrentsSuspenseQueryOptions,
  useListTorrentsSuspense,
} from "../../../api/gen/query/downloadclients/downloadclients";
import { ensureQuery } from "../../../api/query-client";
import { ensureLibrary } from "../../../library/projection";
import { loaderTitle } from "../../../shell/titles";
import { TorrentInspector } from "../../../torrents/inspector";
import { useNames } from "../../../torrents/names";
import { Skeleton } from "../../../ui/skeleton";

export const Route = createFileRoute("/_app/activity/torrents/$torrentId")({
  staticData: { title: loaderTitle },
  loader: async ({ context: { queryClient }, params: { torrentId } }) => {
    const [{ rows }, library] = await Promise.all([
      ensureQuery(queryClient, getListTorrentsSuspenseQueryOptions()),
      ensureLibrary(queryClient),
    ]);
    const row = rows.find(({ id }) => id === torrentId);
    if (row === undefined) {
      // oxlint-disable-next-line typescript/only-throw-error -- the router renders its own notFound() as the not-found page
      throw notFound();
    }
    return { title: library.rows.find(({ id }) => id === row.workId)?.label ?? row.name };
  },
  pendingComponent: () => <Skeleton label="Loading the torrent" />,
  component: TorrentPage,
});

function TorrentPage() {
  const { torrentId } = Route.useParams();
  const navigate = useNavigate();
  const names = useNames();
  const row = useListTorrentsSuspense(undefined, {
    query: { select: ({ rows }) => rows.find(({ id }) => id === torrentId) },
  }).data;
  const close = () => {
    void navigate({ to: "/activity/torrents", search: (prev) => prev, replace: row === undefined });
  };
  useEffect(() => {
    if (row === undefined) {
      close();
    }
  });
  return row === undefined ? null : <TorrentInspector row={row} names={names} onClose={close} />;
}
