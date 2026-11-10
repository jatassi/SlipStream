import { sentenceOf } from "../api/errors";
import { useRemoveFromBlocklist } from "../api/gen/query/grabs/grabs";
import { Button } from "../ui/button";
import { toast } from "../ui/toast";

const releases = (count: number) => (count === 1 ? "Release" : "Releases");

export function BulkBar({ picked, onClear }: { picked: readonly string[]; onClear: () => void }) {
  const remove = useRemoveFromBlocklist({
    mutation: {
      onSuccess: () => {
        toast(`Removed ${picked.length} ${releases(picked.length)} from the Blocklist.`);
        onClear();
      },
      onError: (error) => {
        toast(sentenceOf(error), { kind: "fail" });
      },
    },
  });
  return (
    <div
      role="toolbar"
      aria-label="Picked entries"
      className="mb-4 flex flex-wrap items-center gap-2 rounded-card bg-glass px-4 py-2 ring-1 ring-hairline ring-inset"
    >
      <span className="mr-auto text-ink-1">
        {picked.length} {releases(picked.length)} picked
      </span>
      <Button
        busy={remove.isPending}
        onClick={() => {
          remove.mutate({ data: { infoHashes: [...picked] } });
        }}
      >
        Remove
      </Button>
      <Button look="ghost" onClick={onClear}>
        Clear
      </Button>
    </div>
  );
}
