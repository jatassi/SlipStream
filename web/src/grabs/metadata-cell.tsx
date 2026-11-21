import type { GrabsMetadataSearch } from "../api/gen/model";
import { countWords } from "../format";
import { useClock } from "../ui/clock";
import { whenWords } from "../ui/time-words";

const peer = { one: "peer", many: "peers" };

export function MetadataCell({ search }: { search: GrabsMetadataSearch }) {
  const clock = useClock();
  return (
    <span className="flex max-w-[240px] flex-col gap-0.5">
      <span>
        {search.peers === 0
          ? "Looking for peers"
          : `Getting metadata from ${countWords(search.peers, peer)}`}
      </span>
      <span className="text-meta text-ink-3">
        Fails {whenWords(search.endsAt, "at", clock)} if no peer sends the metadata
      </span>
    </span>
  );
}
