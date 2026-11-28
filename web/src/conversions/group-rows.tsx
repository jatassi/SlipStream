import { useInfiniteQuery } from "@tanstack/react-query";

import type { ConversionsConversionGroup } from "../api/gen/model";
import {
  getConversionGroup,
  getGetConversionGroupQueryKey,
} from "../api/gen/query/conversions/conversions";
import { Button } from "../ui/button";
import { Spinner } from "../ui/spinner";
import { WaitingRow } from "./waiting-row";

export function GroupRows({ group }: { group: ConversionsConversionGroup }) {
  const { data, hasNextPage, fetchNextPage, isFetchingNextPage } = useInfiniteQuery({
    queryKey: [...getGetConversionGroupQueryKey(group.id), "pages"],
    queryFn: ({ pageParam, signal }) =>
      getConversionGroup(group.id, pageParam === undefined ? {} : { cursor: pageParam }, {
        signal,
      }),
    initialPageParam: undefined as string | undefined,
    getNextPageParam: (page) => page.cursor ?? undefined,
  });
  if (data === undefined) {
    return <Spinner />;
  }
  return (
    <ul aria-label={group.label} className="flex flex-col gap-1.5 pl-6">
      {data.pages
        .flatMap((page) => page.rows)
        .map((row) => (
          <li key={row.id}>
            <WaitingRow row={row} />
          </li>
        ))}
      {hasNextPage && (
        <li>
          <Button
            look="ghost"
            busy={isFetchingNextPage}
            onClick={() => {
              void fetchNextPage();
            }}
          >
            Show more
          </Button>
        </li>
      )}
    </ul>
  );
}
