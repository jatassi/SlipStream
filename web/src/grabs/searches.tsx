import type { GrabsSearch } from "../api/gen/model";
import { useListSearchesSuspense } from "../api/gen/query/grabs/grabs";
import { clock } from "../format";
import { Data } from "../ui/data";
import { List, ListRow } from "../ui/list";
import { Section } from "../ui/section";
import { Spinner } from "../ui/spinner";
import { useWaiting } from "./waiting";

const outcomeText: Record<NonNullable<GrabsSearch["outcome"]>, string> = {
  grabbed: "Grabbed a release",
  nothing_acceptable: "Found nothing acceptable",
  indexer_unavailable: "The Indexer didn't answer",
  indexer_refused: "The Indexer refused the search",
};

const found = (count: number) => (count === 1 ? "1 release found" : `${count} releases found`);

function SearchRow({ search, waiting }: { search: GrabsSearch; waiting: string | undefined }) {
  const time = (
    <span className="text-meta">
      <Data>{clock(search.startedAt)}</Data>
    </span>
  );
  if (search.outcome === null) {
    return (
      <ListRow
        title="Searching"
        trailing={
          <span className="flex items-center gap-3">
            <Spinner />
            {time}
          </span>
        }
      />
    );
  }
  return (
    <ListRow
      title={outcomeText[search.outcome]}
      meta={
        search.outcome === "indexer_unavailable"
          ? waiting
          : search.outcome === "indexer_refused"
            ? undefined
            : found(search.candidates)
      }
      trailing={time}
    />
  );
}

export function Searches({ workId }: { workId: string }) {
  const { searches } = useListSearchesSuspense(workId).data;
  const waiting = useWaiting();
  return (
    <Section title="Searches" count={searches.length}>
      {searches.length === 0 ? (
        <p className="text-ink-3">No searches yet. Search looks for a release on every Indexer.</p>
      ) : (
        <List label="Searches">
          {searches.map((search) => (
            <SearchRow key={search.id} search={search} waiting={waiting.get(search.indexerId)} />
          ))}
        </List>
      )}
    </Section>
  );
}
