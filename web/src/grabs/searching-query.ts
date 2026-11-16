import { type QueryClient, useQueryClient } from "@tanstack/react-query";

import {
  getGetWorkSearchingQueryKey,
  getWorkSearching,
  useGetWorkSearchingSuspense,
} from "../api/gen/query/grabs/grabs";

/** What the search queue is doing for a Work. The live effect reads it again on each queue change. */
export const useWorkSearching = (workId: string) =>
  useGetWorkSearchingSuspense(workId, { query: { staleTime: Infinity } }).data;

/** The Work's state read now, past the cache, as a press's toast needs it. */
export const readSearchingNow = (queryClient: QueryClient, workId: string) =>
  queryClient.query({
    queryKey: getGetWorkSearchingQueryKey(workId),
    queryFn: ({ signal }) => getWorkSearching(workId, { signal }),
    staleTime: 0,
  });

export const useReadSearchingNow = () => {
  const queryClient = useQueryClient();
  return (workId: string) => readSearchingNow(queryClient, workId);
};

/**
 * The search group key the server writes for a level, the same as
 * indexers.LevelGroup's. The wire's group carries no level id, so a level
 * header finds its group by this key.
 */
export const levelGroupKey = (levelId: string) => `l:${levelId.replaceAll("-", "").toLowerCase()}`;
