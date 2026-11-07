import type { GrabsSearch } from "../api/gen/model";
import { useListSearchesSuspense } from "../api/gen/query/grabs/grabs";
import { clock } from "../format";
import { Data } from "../ui/data";
import { List, ListRow } from "../ui/list";
import { Section } from "../ui/section";
import { Spinner } from "../ui/spinner";

const outcomeText: Record<NonNullable<GrabsSearch["outcome"]>, string> = {
  grabbed: "Grabbed a Release",
  nothing_acceptable: "Found nothing acceptable",
};

const found = (count: number) => (count === 1 ? "1 Release found" : `${count} Releases found`);

function IndexerLines({ search }: { search: GrabsSearch }) {
  return (
    <ul aria-label="Indexers" className="flex flex-col gap-0.5 py-1 text-meta text-ink-3">
      {search.indexers.map((line, at) => (
        <li key={line.indexerId ?? at} className="flex flex-wrap gap-x-2">
          <span className="text-ink-2">{line.name}</span>
          <span>{line.sentence}</span>
        </li>
      ))}
    </ul>
  );
}

function SearchRow({ search }: { search: GrabsSearch }) {
  const time = (
    <span className="text-meta">
      <Data>{clock(search.startedAt)}</Data>
    </span>
  );
  return (
    <ListRow
      title={search.outcome === null ? "Searching" : outcomeText[search.outcome]}
      meta={search.outcome === null ? undefined : found(search.candidates)}
      detail={search.indexers.length === 0 ? undefined : <IndexerLines search={search} />}
      trailing={
        search.outcome === null ? (
          <span className="flex items-center gap-3">
            <Spinner />
            {time}
          </span>
        ) : (
          time
        )
      }
    />
  );
}

export function Searches({ workId }: { workId: string }) {
  const { searches } = useListSearchesSuspense(workId).data;
  return (
    <Section title="Searches" count={searches.length}>
      {searches.length === 0 ? (
        <p className="text-ink-3">No searches yet. Search looks for a Release on every Indexer.</p>
      ) : (
        <List label="Searches">
          {searches.map((search) => (
            <SearchRow key={search.id} search={search} />
          ))}
        </List>
      )}
    </Section>
  );
}
