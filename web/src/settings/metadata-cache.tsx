import { isApiError } from "../api/errors";
import { useClearMetadataCache } from "../api/gen/query/metadata/metadata";
import { Button } from "../ui/button";
import { toast } from "../ui/toast";

export function ClearCache() {
  const clear = useClearMetadataCache({
    mutation: {
      onSuccess: () => {
        toast("Cleared the metadata cache.");
      },
      onError: (error) => {
        toast(isApiError(error) ? (error.detail ?? error.title) : error.message, { kind: "fail" });
      },
    },
  });
  return (
    <div className="flex flex-col items-start gap-2">
      <p>
        Each Node reading metadata keeps recent answers. Clear them to have every Node ask the
        providers again on its next call.
      </p>
      <Button
        busy={clear.isPending}
        onClick={() => {
          clear.mutate();
        }}
      >
        Clear metadata cache
      </Button>
    </div>
  );
}
