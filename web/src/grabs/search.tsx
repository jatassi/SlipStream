import { Search01Icon } from "@hugeicons/core-free-icons";

import type { GrabsSearchScope, WorksContentType } from "../api/gen/model";
import { useSearchWork } from "../api/gen/query/grabs/grabs";
import { type Noun, countWords } from "../format";
import { Button } from "../ui/button";
import { toast } from "../ui/toast";
import { lowered } from "../works/monitoring/labels";

/** "Searching 12 episodes" when the Items are counted by noun, else "Search started". */
export const startedWords = (count: number, item: Noun | undefined): string =>
  item === undefined ? "Search started" : `Searching ${countWords(count, lowered(item))}`;

/** Starts a search of a Work, or of the levels and Items a scope names; the toast counts by item. */
export function useStartSearch(item?: Noun) {
  const search = useSearchWork({
    mutation: {
      onSuccess: ({ searchIds }) => {
        toast(startedWords(searchIds.length, item));
      },
      onError: (error) => {
        toast(error.message, { kind: "fail" });
      },
    },
  });
  return {
    isPending: search.isPending,
    start: (workId: string, scope?: GrabsSearchScope) => {
      search.mutate(scope === undefined ? { workId } : { workId, data: scope });
    },
  };
}

/** The hero's Search: every wanted Item, counted when the Work has many. */
export function SearchAction({ workId, type }: { workId: string; type: WorksContentType }) {
  const search = useStartSearch(type.items === "many" ? type.item : undefined);
  return (
    <Button
      look="primary"
      icon={Search01Icon}
      busy={search.isPending}
      onClick={() => {
        search.start(workId);
      }}
    >
      Search
    </Button>
  );
}
