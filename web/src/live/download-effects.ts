import { getGetDownloadClientSettingsQueryKey } from "../api/gen/query/downloadclients/downloadclients";
import { getGetTopologyQueryKey } from "../api/gen/query/nodes/nodes";
import type { LiveEffects } from "./types";

export const downloadEffects = {
  "torrents.changed": (_data, { refresh }) => {
    refresh({ queryKey: getGetTopologyQueryKey() });
  },
  "torrents.live": () => {},
  "download-clients.changed": (data, { refresh }) => {
    for (const nodeId of data.nodeIds) {
      refresh({ queryKey: getGetDownloadClientSettingsQueryKey(nodeId) });
    }
  },
} satisfies Pick<LiveEffects, "torrents.changed" | "torrents.live" | "download-clients.changed">;
