import type { ShellCounts } from "../api/gen/model";
import {
  getGetDownloadClientSettingsQueryKey,
  getGetTorrentQueryKey,
  getListDownloadClientsQueryKey,
} from "../api/gen/query/downloadclients/downloadclients";
import { getGetTopologyQueryKey } from "../api/gen/query/nodes/nodes";
import { getGetCountsQueryKey } from "../api/gen/query/shell/shell";
import { spliceTorrents } from "./torrents-splice";
import { takeTorrents } from "./torrents";
import type { LiveEffects } from "./types";

export const downloadEffects = {
  "torrents.changed": (data, { queryClient, refresh }) => {
    void spliceTorrents(queryClient, data.ids);
    for (const id of data.ids) {
      refresh({ queryKey: getGetTorrentQueryKey(id), exact: true });
    }
    refresh({ queryKey: getGetTopologyQueryKey() });
    refresh({ queryKey: getGetCountsQueryKey() });
    refresh({ queryKey: getListDownloadClientsQueryKey() });
  },
  "torrents.live": (data, { queryClient }) => {
    takeTorrents(queryClient, data);
    queryClient.setQueryData<ShellCounts>(
      getGetCountsQueryKey(),
      (counts) => counts && { ...counts, downloads: data.downloads },
    );
  },
  "download-clients.changed": (data, { refresh }) => {
    refresh({ queryKey: getListDownloadClientsQueryKey() });
    for (const nodeId of data.nodeIds) {
      refresh({ queryKey: getGetDownloadClientSettingsQueryKey(nodeId) });
    }
  },
} satisfies Pick<LiveEffects, "torrents.changed" | "torrents.live" | "download-clients.changed">;
