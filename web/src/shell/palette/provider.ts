import { keepPreviousData, useQueries, useQuery } from "@tanstack/react-query";
import { useEffect, useState } from "react";

import type { WorksCandidate, WorksContentType, WorksLibraryRow } from "../../api/gen/model";
import {
  getSearchCatalogQueryKey,
  getSearchWorksQueryKey,
  searchCatalog,
  searchWorks,
} from "../../api/gen/query/works/works";

const worksShown = 8;
const catalogShown = 5;
const catalogPause = 400;
const catalogShortest = 2;

export function useFoundWorks(query: string): readonly WorksLibraryRow[] {
  const q = query.trim();
  const params = { q, limit: worksShown };
  const found = useQuery({
    queryKey: getSearchWorksQueryKey(params),
    queryFn: ({ signal }) => searchWorks(params, { signal }),
    enabled: q !== "",
    placeholderData: keepPreviousData,
  }).data;
  return q === "" ? [] : (found?.rows ?? []);
}

function useSettled(value: string, delay: number): string {
  const [settled, setSettled] = useState(value);
  useEffect(() => {
    const timer = setTimeout(() => {
      setSettled(value);
    }, delay);
    return () => {
      clearTimeout(timer);
    };
  }, [value, delay]);
  return settled;
}

export type CatalogResults = {
  readonly type: WorksContentType;
  readonly query: string;
  readonly candidates: readonly WorksCandidate[];
};

/**
 * Each type's first catalog results for `query`, asked once typing pauses
 * with at least two characters, so a burst of keys costs the metadata
 * providers one search.
 */
export function useCatalogResults(
  query: string,
  types: readonly WorksContentType[],
): readonly CatalogResults[] {
  const settled = useSettled(query.trim(), catalogPause);
  const asking = settled.length >= catalogShortest;
  const answers = useQueries({
    queries: types.map((type) => ({
      queryKey: getSearchCatalogQueryKey(type.id, { query: settled }),
      queryFn: ({ signal }: { signal: AbortSignal }) =>
        searchCatalog(type.id, { query: settled }, { signal }),
      enabled: asking,
      placeholderData: keepPreviousData,
    })),
  });
  if (!asking) {
    return [];
  }
  return types.flatMap((type, index) => {
    const candidates = answers[index]?.data?.candidates;
    return candidates === undefined
      ? []
      : [{ type, query: settled, candidates: candidates.slice(0, catalogShown) }];
  });
}
