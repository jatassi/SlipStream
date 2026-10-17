import { getGetWorkCellsQueryKey } from "../api/gen/query/files/files";
import {
  getGetGrabQueryKey,
  getListGrabsQueryKey,
  getListSearchesQueryKey,
} from "../api/gen/query/grabs/grabs";
import {
  getGetAccountQueryKey,
  getGetSessionQueryKey,
  getListAccountSessionsQueryKey,
  getListAccountsQueryKey,
  getListPasskeysQueryKey,
  getListSessionsQueryKey,
  getListSignupLinksQueryKey,
} from "../api/gen/query/identity/identity";
import { getListIndexersQueryKey } from "../api/gen/query/indexers/indexers";
import { getListRootsQueryKey } from "../api/gen/query/roots/roots";
import { getGetCountsQueryKey } from "../api/gen/query/shell/shell";
import { getListProfilesQueryKey, getListTargetsQueryKey } from "../api/gen/query/targets/targets";
import {
  getGetWorkQueryKey,
  getListContentTypesQueryKey,
  getListWorksQueryKey,
} from "../api/gen/query/works/works";
import { systemEffects } from "./system-effects";
import type { LiveEffects } from "./types";

const everyPath = (pattern: RegExp) => ({
  predicate: ({ queryKey: [path] }: { queryKey: readonly unknown[] }) =>
    typeof path === "string" && pattern.test(path),
});
const everyWorksCells = everyPath(/^\/api\/v1\/works\/[^/]+\/cells$/u);
const everyAccount = everyPath(/^\/api\/v1\/accounts\/[^/]+(\/sessions)?$/u);
const everySignupLink = everyPath(/^\/api\/v1\/signup-links\/[^/]+$/u);
const everyTarget = everyPath(/^\/api\/v1\/targets\/[^/]+$/u);
const everyCatalogSearch = everyPath(/^\/api\/v1\/content-types\/[^/]+\/catalog$/u);

export const effects = {
  ...systemEffects,
  "identity.sessions-changed": (data, { queryClient }) => {
    void queryClient.invalidateQueries({ queryKey: getGetSessionQueryKey() });
    void queryClient.invalidateQueries({ queryKey: getListSessionsQueryKey() });
    void queryClient.invalidateQueries({ queryKey: getListAccountsQueryKey() });
    void queryClient.invalidateQueries({ queryKey: getGetAccountQueryKey(data.account) });
    void queryClient.invalidateQueries({ queryKey: getListAccountSessionsQueryKey(data.account) });
  },
  "identity.accounts-changed": (_data, { queryClient }) => {
    void queryClient.invalidateQueries({ queryKey: getGetSessionQueryKey() });
    void queryClient.invalidateQueries({ queryKey: getListAccountsQueryKey() });
    void queryClient.invalidateQueries(everyAccount);
  },
  "identity.passkeys-changed": (_data, { queryClient }) => {
    void queryClient.invalidateQueries({ queryKey: getListPasskeysQueryKey() });
  },
  "identity.links-changed": (_data, { queryClient }) => {
    void queryClient.invalidateQueries({ queryKey: getListSignupLinksQueryKey() });
    void queryClient.invalidateQueries(everySignupLink);
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
    void queryClient.invalidateQueries(everyTarget);
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
  "grabs.changed": (data, { queryClient }) => {
    void queryClient.invalidateQueries({ queryKey: getListGrabsQueryKey() });
    void queryClient.invalidateQueries({ queryKey: getGetGrabQueryKey(data.grabId) });
    void queryClient.invalidateQueries({ queryKey: getGetWorkCellsQueryKey(data.workId) });
    void queryClient.invalidateQueries({ queryKey: getGetCountsQueryKey() });
  },
  "grabs.search-changed": (data, { queryClient }) => {
    void queryClient.invalidateQueries({ queryKey: getListSearchesQueryKey(data.workId) });
  },
} satisfies LiveEffects;
