import { Search01Icon } from "@hugeicons/core-free-icons";

import type { GrabsSearchQueued, GrabsSearchScope } from "../api/gen/model";
import { useSearchWork } from "../api/gen/query/grabs/grabs";
import { type Noun, countWords } from "../format";
import { type ManualScope, openManualSearch } from "../manual/sheet-state";
import { Button, type ButtonLook, type ButtonSize } from "../ui/button";
import { toast } from "../ui/toast";
import { lowered } from "../works/monitoring/labels";
import { queuedToast } from "./search-words";
import { useReadSearchingNow } from "./searching-query";

/** What a press of Search names in its toast: a count of the Items it queued by noun, or a Work's or Item's name. */
export type Searched = { readonly item: Noun } | { readonly name: string };

const searchedWords = (searched: Searched, answer: GrabsSearchQueued): string =>
  "name" in searched
    ? searched.name
    : countWords(
        new Set(answer.groups.flatMap(({ itemIds }) => itemIds)).size,
        lowered(searched.item),
      );

/** Queues a search of a Work, or of the levels and Items a scope names, and toasts what it queued. */
export function useStartSearch() {
  const readNow = useReadSearchingNow();
  const announce = async (workId: string, answer: GrabsSearchQueued, searched: Searched) => {
    toast(queuedToast(answer, await readNow(workId), searchedWords(searched, answer)));
  };
  const search = useSearchWork({
    mutation: {
      onError: (error) => {
        toast(
          error.message,
          "code" in error && error.code === "nothing_wanted" ? {} : { kind: "fail" },
        );
      },
    },
  });
  return {
    isPending: search.isPending,
    start: (workId: string, searched: Searched, scope?: GrabsSearchScope) => {
      search.mutate(scope === undefined ? { workId } : { workId, data: scope }, {
        onSuccess: (answer) => {
          void announce(workId, answer, searched);
        },
      });
    },
  };
}

/** The hero's Search: every wanted Item of the Work. */
export function SearchAction({ workId, searched }: { workId: string; searched: Searched }) {
  const search = useStartSearch();
  return (
    <Button
      look="primary"
      icon={Search01Icon}
      busy={search.isPending}
      onClick={() => {
        search.start(workId, searched);
      }}
    >
      Search
    </Button>
  );
}

const wholeWork: ManualScope = {};

/** Opens Manual search on a Work, or on the part of it a scope names. */
export function ManualSearchButton({
  workId,
  scope = wholeWork,
  label,
  size = "default",
  look = "default",
}: {
  workId: string;
  scope?: ManualScope;
  label?: string;
  size?: ButtonSize;
  look?: ButtonLook;
}) {
  return (
    <Button
      size={size}
      look={look}
      aria-label={label}
      onClick={() => {
        openManualSearch({ workId, scope });
      }}
    >
      Manual search
    </Button>
  );
}
