import { keepPreviousData, useQuery } from "@tanstack/react-query";
import { useEffect, useState } from "react";

import type { WorksContentType } from "../api/gen/model";
import { getSearchWorksQueryKey, searchWorks } from "../api/gen/query/works/works";

const pause = 200;

const idsOf = ({ workIds }: { workIds: readonly string[] }): ReadonlySet<string> =>
  new Set(workIds);

/**
 * The Works the search box matches by any title they are known by, read
 * from the server once typing pauses. Undefined while the box is empty.
 */
export function useMatched(
  q: string,
  types: readonly WorksContentType[],
): ReadonlySet<string> | undefined {
  const [settled, setSettled] = useState(q);
  useEffect(() => {
    const timer = setTimeout(() => {
      setSettled(q);
    }, pause);
    return () => {
      clearTimeout(timer);
    };
  }, [q]);
  const [only] = types.length === 1 ? types : [];
  const params = {
    q: settled.trim(),
    limit: 0,
    ...(only !== undefined && { contentType: only.id }),
  };
  const { data } = useQuery({
    queryKey: getSearchWorksQueryKey(params),
    queryFn: ({ signal }) => searchWorks(params, { signal }),
    enabled: params.q !== "",
    placeholderData: keepPreviousData,
    select: idsOf,
  });
  return q.trim() === "" ? undefined : data;
}
