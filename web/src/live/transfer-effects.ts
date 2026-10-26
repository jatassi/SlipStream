import {
  getGetTransferQueryKey,
  getGetTransferSettingsQueryKey,
  getListTransferSpeedsQueryKey,
  getListTransfersQueryKey,
} from "../api/gen/query/transfers/transfers";
import type { LiveEffects } from "./types";

export const transferEffects = {
  "transfers.changed": (data, { refresh }) => {
    refresh({ queryKey: getListTransfersQueryKey() });
    refresh({ queryKey: getListTransferSpeedsQueryKey() });
    for (const id of data.transferIds) {
      refresh({ queryKey: getGetTransferQueryKey(id) });
    }
    for (const nodeId of data.nodeIds) {
      refresh({ queryKey: getGetTransferSettingsQueryKey(nodeId) });
    }
  },
} satisfies Pick<LiveEffects, "transfers.changed">;
