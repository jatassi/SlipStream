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
import { getListMetadataProvidersQueryKey } from "../api/gen/query/metadata/metadata";
import { downloadEffects } from "./download-effects";
import { indexerEffects } from "./indexer-effects";
import { libraryEffects, renameCounts } from "./library-effects";
import { mediaServerEffects } from "./media-server-effects";
import { everyPath } from "./refresh";
import { systemEffects } from "./system-effects";
import { targetsEffects } from "./targets-effects";
// oxlint-disable-next-line import/max-dependencies -- every product event's effects meet in this one table
import type { LiveEffects } from "./types";

const everyAccount = everyPath(/^\/api\/v1\/accounts\/[^/]+(\/sessions)?$/u);
const everySignupLink = everyPath(/^\/api\/v1\/signup-links\/[^/]+$/u);
const everyNaming = everyPath(/^\/api\/v1\/naming(\/|$)/u);

export const effects = {
  ...systemEffects,
  ...libraryEffects,
  ...downloadEffects,
  ...indexerEffects,
  ...targetsEffects,
  ...mediaServerEffects,
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
  "metadata.changed": (_data, { refresh }) => {
    refresh({ queryKey: getListMetadataProvidersQueryKey() });
  },
  "naming.changed": (_data, { refresh }) => {
    refresh(everyNaming);
    refresh(renameCounts);
  },
} satisfies LiveEffects;
