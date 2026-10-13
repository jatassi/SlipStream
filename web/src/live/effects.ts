import { getGetSessionQueryKey } from "../api/gen/query/identity/identity";
import { getGetNodeQueryKey, getListNodesQueryKey } from "../api/gen/query/nodes/nodes";
import {
  getGetWorkQueryKey,
  getListContentTypesQueryKey,
  getListWorksQueryKey,
} from "../api/gen/query/works/works";
import type { LiveEffects } from "./types";

export const effects = {
  "identity.sessions-changed": (_data, { queryClient }) => {
    void queryClient.invalidateQueries({ queryKey: getGetSessionQueryKey() });
  },
  "nodes.changed": (data, { queryClient }) => {
    void queryClient.invalidateQueries({ queryKey: getListNodesQueryKey() });
    void queryClient.invalidateQueries({ queryKey: getGetNodeQueryKey(data.nodeId) });
  },
  "works.changed": (data, { queryClient }) => {
    void queryClient.invalidateQueries({ queryKey: getListWorksQueryKey() });
    void queryClient.invalidateQueries({ queryKey: getGetWorkQueryKey(data.workId) });
  },
  "works.content-types-changed": (_data, { queryClient }) => {
    void queryClient.invalidateQueries({ queryKey: getListContentTypesQueryKey() });
  },
} satisfies LiveEffects;
