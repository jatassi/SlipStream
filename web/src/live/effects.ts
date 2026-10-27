import { getGetFirstRunQueryKey } from "../api/gen/query/firstrun/firstrun";
import {
  getGetAccountQueryKey,
  getGetAddressQueryKey,
  getGetSessionQueryKey,
  getListAccountSessionsQueryKey,
  getListAccountsQueryKey,
  getListPasskeysQueryKey,
  getListSessionsQueryKey,
  getListSignupLinksQueryKey,
} from "../api/gen/query/identity/identity";
import {
  getGetIndexerCatalogQueryKey,
  getGetIndexerQueryKey,
  getListIndexersQueryKey,
} from "../api/gen/query/indexers/indexers";
import { getListMetadataProvidersQueryKey } from "../api/gen/query/metadata/metadata";
import {
  getListCutsQueryKey,
  getListProfilesQueryKey,
  getListTargetsQueryKey,
} from "../api/gen/query/targets/targets";
import { downloadEffects } from "./download-effects";
import { libraryEffects } from "./library-effects";
import { everyPath } from "./refresh";
import { systemEffects } from "./system-effects";
import type { LiveEffects } from "./types";

const everyWorksCells = everyPath(/^\/api\/v1\/works\/[^/]+\/cells$/u);
const everyAccount = everyPath(/^\/api\/v1\/accounts\/[^/]+(\/sessions)?$/u);
const everySignupLink = everyPath(/^\/api\/v1\/signup-links\/[^/]+$/u);
const everyTarget = everyPath(/^\/api\/v1\/targets\/[^/]+$/u);
const everyProfile = everyPath(/^\/api\/v1\/profiles\/[^/]+$/u);
const everyWorksWants = everyPath(/^\/api\/v1\/works\/[^/]+\/wants$/u);
const everyCatalogDefinition = everyPath(/^\/api\/v1\/indexer-catalog\/[^/]+$/u);

export const effects = {
  ...systemEffects,
  ...libraryEffects,
  ...downloadEffects,
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
  "identity.address-changed": (_data, { refresh }) => {
    refresh({ queryKey: getGetAddressQueryKey() });
  },
  "firstrun.changed": (_data, { refresh }) => {
    refresh({ queryKey: getGetFirstRunQueryKey() });
  },
  "identity.links-changed": (_data, { refresh }) => {
    refresh({ queryKey: getListSignupLinksQueryKey() });
    refresh(everySignupLink);
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
  "indexers.changed": (_data, { refresh }) => {
    refresh({ queryKey: getListIndexersQueryKey() });
  },
  "indexers.status-changed": (data, { refresh }) => {
    refresh({ queryKey: getListIndexersQueryKey() });
    refresh({ queryKey: getGetIndexerQueryKey(data.indexerId) });
  },
  "indexers.catalog-changed": (_data, { refresh }) => {
    refresh({ queryKey: getGetIndexerCatalogQueryKey() });
    refresh(everyCatalogDefinition);
  },
  "metadata.changed": (_data, { refresh }) => {
    refresh({ queryKey: getListMetadataProvidersQueryKey() });
  },
} satisfies LiveEffects;
