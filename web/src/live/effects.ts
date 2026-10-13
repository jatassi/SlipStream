import { getGetWorkCellsQueryKey } from "../api/gen/query/files/files";
import { getGetSessionQueryKey } from "../api/gen/query/identity/identity";
import { getGetNodeQueryKey, getListNodesQueryKey } from "../api/gen/query/nodes/nodes";
import { getListRootsQueryKey } from "../api/gen/query/roots/roots";
import { getListProfilesQueryKey, getListTargetsQueryKey } from "../api/gen/query/targets/targets";
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
  "targets.changed": (_data, { queryClient }) => {
    void queryClient.invalidateQueries({ queryKey: getListTargetsQueryKey() });
    void queryClient.invalidateQueries({ queryKey: getListProfilesQueryKey() });
  },
  "roots.changed": (_data, { queryClient }) => {
    void queryClient.invalidateQueries({ queryKey: getListRootsQueryKey() });
  },
  "files.changed": (data, { queryClient }) => {
    void queryClient.invalidateQueries({ queryKey: getGetWorkCellsQueryKey(data.workId) });
  },
} satisfies LiveEffects;
