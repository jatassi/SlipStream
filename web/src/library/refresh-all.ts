import { sentenceOf } from "../api/errors";
import type { WorksContentType } from "../api/gen/model";
import { useRefreshContentType } from "../api/gen/query/metadata/metadata";
import { toast } from "../ui/toast";

/** Refreshes every Work of the one type a Library shows; "All" has no such action. */
export function useRefreshAll(types: readonly WorksContentType[]): (() => void) | undefined {
  const refresh = useRefreshContentType();
  const [only] = types.length === 1 ? types : [];
  return (
    only &&
    (() => {
      refresh.mutate(
        { contentType: only.id },
        {
          onSuccess: () => {
            toast(`Refreshing the metadata of every ${only.work.one}.`);
          },
          onError: (error) => {
            toast(sentenceOf(error), { kind: "fail" });
          },
        },
      );
    })
  );
}
