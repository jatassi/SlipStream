import {
  getGetDownloadClientSettingsQueryKey,
  getListDownloadClientsQueryKey,
} from "../api/gen/query/downloadclients/downloadclients";
import { getGetTopologyQueryKey } from "../api/gen/query/nodes/nodes";
import { getGetCountsQueryKey } from "../api/gen/query/shell/shell";
import type { LiveEffects } from "./types";

export const downloadEffects = {
  "torrents.changed": (_data, { refresh }) => {
    refresh({ queryKey: getGetTopologyQueryKey() });
    refresh({ queryKey: getGetCountsQueryKey() });
    refresh({ queryKey: getListDownloadClientsQueryKey() });
  },
  "torrents.live": () => {},
  "download-clients.changed": (data, { refresh }) => {
    refresh({ queryKey: getListDownloadClientsQueryKey() });
    for (const nodeId of data.nodeIds) {
      refresh({ queryKey: getGetDownloadClientSettingsQueryKey(nodeId) });
    }
  },
} satisfies Pick<LiveEffects, "torrents.changed" | "torrents.live" | "download-clients.changed">;
