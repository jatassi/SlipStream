import {
  getGetTorrentQueryKey,
  getListTorrentsQueryKey,
  listTorrents,
} from "../api/gen/query/downloadclients/downloadclients";
import type { DownloadclientsTorrentList } from "../api/gen/model";
import { refresh } from "./refresh";
import { createSplicer } from "./splice";

/**
 * Splices the named Torrents into the cached list, then reads again the
 * detail of each one still listed; a removed one's detail is gone.
 */
export const torrentsSplicer = createSplicer<DownloadclientsTorrentList>({
  queryKey: getListTorrentsQueryKey(),
  read: (ids) => listTorrents({ ids: [...ids] }),
  landed: (queryClient, ids) => {
    const listed = new Set(
      queryClient
        .getQueryData<DownloadclientsTorrentList>(getListTorrentsQueryKey())
        ?.rows.map(({ id }) => id),
    );
    for (const id of ids.filter((named) => listed.has(named))) {
      void refresh(queryClient, { queryKey: getGetTorrentQueryKey(id), exact: true });
    }
  },
});
