import type { GrabsMetadataSearch } from "../api/gen/model";
import { countWords, sinceWords } from "../format";
import { useNow } from "../tasks/use-now";
import { Data } from "../ui/data";

const peer = { one: "peer", many: "peers" };

export function MetadataCell({ search }: { search: GrabsMetadataSearch }) {
  const reading = { now: useNow(60_000), zone: undefined };
  return (
    <span className="flex max-w-[240px] flex-col gap-0.5">
      <span>
        {search.peers === 0
          ? "Looking for peers"
          : `Getting metadata from ${countWords(search.peers, peer)}`}
      </span>
      <span className="text-meta text-ink-3">
        Fails at <Data>{sinceWords(search.endsAt, reading)}</Data> if no peer sends the metadata
      </span>
    </span>
  );
}
