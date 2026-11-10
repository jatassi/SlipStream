import type { QueryClient } from "@tanstack/react-query";

import {
  getGetIndexerCatalogSuspenseQueryOptions,
  getGetIndexerSettingsSuspenseQueryOptions,
} from "../api/gen/query/indexers/indexers";
import { getListTasksSuspenseQueryOptions } from "../api/gen/query/tasks/tasks";
import { ensureQuery, freshQuery } from "../api/query-client";
import { CatalogSection } from "./catalog/catalog-section";
import { LongBackoff } from "./long-backoff";
import { Searching } from "./searching";

/** What the catalog, searching and warning blocks under the Indexers list read. */
export async function loadSectionTail(queryClient: QueryClient): Promise<void> {
  await Promise.all([
    freshQuery(queryClient, getGetIndexerSettingsSuspenseQueryOptions()),
    ensureQuery(queryClient, getGetIndexerCatalogSuspenseQueryOptions()),
    ensureQuery(queryClient, getListTasksSuspenseQueryOptions()),
  ]);
}

/** The catalog, the search wait and the long-backoff warning, under the Indexers list. */
export function SectionTail() {
  return (
    <>
      <CatalogSection />
      <Searching />
      <LongBackoff />
    </>
  );
}
