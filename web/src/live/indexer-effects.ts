import {
  getGetIndexerCatalogQueryKey,
  getGetIndexerQueryKey,
  getGetIndexerSettingsQueryKey,
  getListIndexerQueriesQueryKey,
  getListIndexersQueryKey,
  getGetTrackerQueryKey,
  getListTrackersQueryKey,
} from "../api/gen/query/indexers/indexers";
import { everyPath } from "./refresh";
import type { LiveEffects } from "./types";

const everyIndexer = everyPath(/^\/api\/v1\/indexers\/[^/]+$/u);
const everyCatalogDefinition = everyPath(/^\/api\/v1\/indexer-catalog\/[^/]+$/u);

export const indexerEffects = {
  "indexers.changed": (_data, { refresh }) => {
    refresh({ queryKey: getListIndexersQueryKey() });
    refresh(everyIndexer);
    refresh({ queryKey: getListTrackersQueryKey() });
  },
  "indexers.status-changed": (data, { refresh }) => {
    refresh({ queryKey: getListIndexersQueryKey() });
    refresh({ queryKey: getGetIndexerQueryKey(data.indexerId) });
    refresh({ queryKey: getListIndexerQueriesQueryKey(data.indexerId) });
  },
  "indexers.catalog-changed": (_data, { refresh }) => {
    refresh({ queryKey: getGetIndexerCatalogQueryKey() });
    refresh(everyCatalogDefinition);
  },
  "indexers.settings-changed": (_data, { refresh }) => {
    refresh({ queryKey: getGetIndexerSettingsQueryKey() });
  },
  "trackers.changed": (data, { refresh }) => {
    refresh({ queryKey: getListTrackersQueryKey() });
    refresh({ queryKey: getGetTrackerQueryKey(data.trackerId) });
    refresh({ queryKey: getListIndexersQueryKey() });
    refresh(everyIndexer);
  },
} satisfies Pick<
  LiveEffects,
  | "indexers.changed"
  | "indexers.status-changed"
  | "indexers.catalog-changed"
  | "indexers.settings-changed"
  | "trackers.changed"
>;
