import type { GrabsWantedRow } from "../api/gen/model";
import { useSearchWork } from "../api/gen/query/grabs/grabs";
import { toast } from "../ui/toast";

/** Searches one Work in the now lane, saying when every group of it was already searching. */
export function useRowSearch() {
  const search = useSearchWork();
  return {
    busy: search.isPending ? search.variables.workId : undefined,
    start: (row: GrabsWantedRow) => {
      search.mutate(
        { workId: row.work.id },
        {
          onSuccess: ({ groups }) => {
            const merged = groups.length > 0 && groups.every(({ state }) => state === "merged");
            toast(merged ? `Already searching ${row.work.label}` : `Searching ${row.work.label}`);
          },
          onError: (error) => {
            toast(error.message, { kind: "fail" });
          },
        },
      );
    },
  };
}

export type RowSearch = ReturnType<typeof useRowSearch>;

export const workLink = (row: GrabsWantedRow) =>
  ({ to: "/works/$workId", params: { workId: row.work.id } }) as const;
