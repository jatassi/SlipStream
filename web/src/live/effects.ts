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
import {
  getListCutsQueryKey,
  getListProfilesQueryKey,
  getListTargetsQueryKey,
} from "../api/gen/query/targets/targets";
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
const everyProfile = everyPath(/^\/api\/v1\/profiles\/[^/]+$/u);
const everyWorksWants = everyPath(/^\/api\/v1\/works\/[^/]+\/wants$/u);
const everyCatalogSearch = everyPath(/^\/api\/v1\/content-types\/[^/]+\/catalog$/u);

export const effects = {
  ...systemEffects,
  "identity.sessions-changed": (data, { refresh }) => {
    refresh({ queryKey: getGetSessionQueryKey() });
    refresh({ queryKey: getListSessionsQueryKey() });
    refresh({ queryKey: getListAccountsQueryKey() });
    refresh({ queryKey: getGetAccountQueryKey(data.account) });
    refresh({ queryKey: getListAccountSessionsQueryKey(data.account) });
  },
  "identity.accounts-changed": (_data, { refresh }) => {
    refresh({ queryKey: getGetSessionQueryKey() });
    refresh({ queryKey: getListAccountsQueryKey() });
    refresh(everyAccount);
  },
  "identity.passkeys-changed": (_data, { refresh }) => {
    refresh({ queryKey: getListPasskeysQueryKey() });
  },
  "identity.links-changed": (_data, { refresh }) => {
    refresh({ queryKey: getListSignupLinksQueryKey() });
    refresh(everySignupLink);
  },
  "works.changed": (data, { refresh }) => {
    refresh({ queryKey: getListWorksQueryKey() });
    refresh({ queryKey: getGetWorkQueryKey(data.workId) });
    refresh({ queryKey: getGetWorkCellsQueryKey(data.workId) });
    refresh(everyCatalogSearch);
  },
  "works.content-types-changed": (_data, { refresh }) => {
    refresh({ queryKey: getListContentTypesQueryKey() });
  },
  "targets.changed": (_data, { refresh }) => {
    refresh({ queryKey: getListTargetsQueryKey() });
    refresh(everyTarget);
    refresh({ queryKey: getListProfilesQueryKey() });
    refresh(everyProfile);
    refresh({ queryKey: getListCutsQueryKey() });
    refresh(everyWorksWants);
    refresh(everyWorksCells);
  },
  "roots.changed": (_data, { refresh }) => {
    refresh({ queryKey: getListRootsQueryKey() });
  },
  "indexers.changed": (_data, { refresh }) => {
    refresh({ queryKey: getListIndexersQueryKey() });
  },
  "files.changed": (data, { refresh }) => {
    refresh({ queryKey: getGetWorkCellsQueryKey(data.workId) });
  },
  "grabs.changed": (data, { refresh }) => {
    refresh({ queryKey: getListGrabsQueryKey() });
    refresh({ queryKey: getGetGrabQueryKey(data.grabId) });
    refresh({ queryKey: getGetWorkCellsQueryKey(data.workId) });
    refresh({ queryKey: getGetCountsQueryKey() });
  },
  "grabs.search-changed": (data, { refresh }) => {
    refresh({ queryKey: getListSearchesQueryKey(data.workId) });
  },
} satisfies LiveEffects;
