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
  "metadata.changed": (_data, { refresh }) => {
    refresh({ queryKey: getListMetadataProvidersQueryKey() });
  },
  // Nothing in the app reads Transfers yet.
  "transfers.changed": () => {},
} satisfies LiveEffects;
