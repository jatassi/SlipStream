import type { Query } from "@tanstack/react-query";

import { getGetWorkCellsQueryKey } from "../api/gen/query/files/files";
import { getGetSessionQueryKey } from "../api/gen/query/identity/identity";
import { getListIndexersQueryKey } from "../api/gen/query/indexers/indexers";
import { getGetNodeQueryKey, getListNodesQueryKey } from "../api/gen/query/nodes/nodes";
import { getListRootsQueryKey } from "../api/gen/query/roots/roots";
import { getListProfilesQueryKey, getListTargetsQueryKey } from "../api/gen/query/targets/targets";
import {
  getGetWorkQueryKey,
  getListContentTypesQueryKey,
  getListWorksQueryKey,
} from "../api/gen/query/works/works";
import type { LiveEffects } from "./types";

const everyPath = (pattern: RegExp) => ({
  predicate: ({ queryKey: [path] }: Query) => typeof path === "string" && pattern.test(path),
});
const everyWorksCells = everyPath(/^\/api\/v1\/works\/[^/]+\/cells$/u);
const everyCatalogSearch = everyPath(/^\/api\/v1\/content-types\/[^/]+\/catalog$/u);

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
    void queryClient.invalidateQueries({ queryKey: getGetWorkCellsQueryKey(data.workId) });
    void queryClient.invalidateQueries(everyCatalogSearch);
  },
  "works.content-types-changed": (_data, { queryClient }) => {
    void queryClient.invalidateQueries({ queryKey: getListContentTypesQueryKey() });
  },
  "targets.changed": (_data, { queryClient }) => {
    void queryClient.invalidateQueries({ queryKey: getListTargetsQueryKey() });
    void queryClient.invalidateQueries({ queryKey: getListProfilesQueryKey() });
    void queryClient.invalidateQueries(everyWorksCells);
  },
  "roots.changed": (_data, { queryClient }) => {
    void queryClient.invalidateQueries({ queryKey: getListRootsQueryKey() });
  },
  "indexers.changed": (_data, { queryClient }) => {
    void queryClient.invalidateQueries({ queryKey: getListIndexersQueryKey() });
  },
  "files.changed": (data, { queryClient }) => {
    void queryClient.invalidateQueries({ queryKey: getGetWorkCellsQueryKey(data.workId) });
  },
} satisfies LiveEffects;
