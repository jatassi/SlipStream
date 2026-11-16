import type { IndexersIndexer } from "../api/gen/model";
import { agoWords } from "../format";
import { useNow } from "../tasks/use-now";

const agoTick = 60_000;

/** When RSS sync last read the Indexer's feed, and whether it may have missed Releases since. */
export function RSSFetch({ indexer }: { indexer: IndexersIndexer }) {
  const now = useNow(agoTick);
  const fetch = indexer.rssFetch;
  if (fetch === null) {
    return <span>Not read yet</span>;
  }
  return (
    <span className="flex flex-col gap-1">
      <span>{`Read ${agoWords(fetch.lastFetchAt, now)}`}</span>
      {fetch.gapKind !== null && (
        <span className="text-warn-ink">
          RSS sync may have missed releases. Its Health item says what to do.
        </span>
      )}
    </span>
  );
}
