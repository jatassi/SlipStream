import { Search01Icon } from "@hugeicons/core-free-icons";

import { useSearchWork } from "../api/gen/query/grabs/grabs";
import { Button } from "../ui/button";
import { toast } from "../ui/toast";

export function useStartSearch() {
  return useSearchWork({
    mutation: {
      onSuccess: ({ searchIds }) => {
        toast(
          searchIds.length === 0
            ? "Nothing is wanted, so there's nothing to search for."
            : "Search started",
        );
      },
      onError: (error) => {
        toast(error.message, { kind: "fail" });
      },
    },
  });
}

export function SearchAction({ workId }: { workId: string }) {
  const search = useStartSearch();
  return (
    <Button
      look="primary"
      icon={Search01Icon}
      busy={search.isPending}
      onClick={() => {
        search.mutate({ workId });
      }}
    >
      Search
    </Button>
  );
}
