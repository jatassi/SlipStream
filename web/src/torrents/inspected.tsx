import type { QueryClient } from "@tanstack/react-query";
import { notFound } from "@tanstack/react-router";
import { useEffect } from "react";

import {
  getListTorrentsSuspenseQueryOptions,
  useListTorrentsSuspense,
} from "../api/gen/query/downloadclients/downloadclients";
import { ensureQuery } from "../api/query-client";
import { ensureLibrary } from "../library/projection";
import type { Titled } from "../shell/titles";
import { TorrentInspector } from "./inspector";
import { useNames } from "./names";

/** An inspected Torrent's title: its Work's label, or its release name. */
export async function loadInspected(queryClient: QueryClient, torrentId: string): Promise<Titled> {
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
}

/** The Inspector of one Torrent; it closes itself once the Torrent is gone. */
export function InspectedTorrent({
  torrentId,
  onClose,
}: {
  torrentId: string;
  onClose: (gone: boolean) => void;
}) {
  const names = useNames();
  const row = useListTorrentsSuspense(undefined, {
    query: { select: ({ rows }) => rows.find(({ id }) => id === torrentId) },
  }).data;
  useEffect(() => {
    if (row === undefined) {
      onClose(true);
    }
  });
  return row === undefined ? null : (
    <TorrentInspector
      row={row}
      names={names}
      onClose={() => {
        onClose(false);
      }}
    />
  );
}
