import type { IndexersIndexer } from "../api/gen/model";
import { RelativeTime } from "../ui/relative-time";
import { SignalLine } from "../ui/signal-line";

/** When RSS sync last read the Indexer's feed, and whether it may have missed Releases since. */
export function RSSFetch({ indexer }: { indexer: IndexersIndexer }) {
  const fetch = indexer.rssFetch;
  if (fetch === null) {
    return <span>Not read yet</span>;
  }
  return (
    <div className="flex flex-col gap-1">
      <span>
        Read <RelativeTime at={fetch.lastFetchAt} form="age" />
      </span>
      {fetch.gapKind !== null && (
        <SignalLine tone="warn">
          RSS sync may have missed releases. Its Health item says what to do.
        </SignalLine>
      )}
    </div>
  );
}
