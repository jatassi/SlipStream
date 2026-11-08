import type { GrabsSearch } from "../api/gen/model";
import { useListSearchesSuspense } from "../api/gen/query/grabs/grabs";
import { clock } from "../format";
import { Data } from "../ui/data";
import { List, ListRow } from "../ui/list";
import { Section } from "../ui/section";
import { Spinner } from "../ui/spinner";
import { RunReleases } from "./search-releases";

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

/** Each Item's latest finished run, the only runs whose Releases are kept. */
function latestRuns(searches: readonly GrabsSearch[]): ReadonlySet<string> {
  const latest = new Map<string, string>();
  for (const search of searches) {
    if (search.outcome !== null && !latest.has(search.itemId)) {
      latest.set(search.itemId, search.id);
    }
  }
  return new Set(latest.values());
}

function RunDetails({ search, releases }: { search: GrabsSearch; releases: boolean }) {
  return (
    <>
      {search.indexers.length > 0 && <IndexerLines search={search} />}
      {releases && <RunReleases search={search} />}
    </>
  );
}

function SearchRow({ search, releases }: { search: GrabsSearch; releases: boolean }) {
  const time = (
    <span className="text-meta">
      <Data>{clock(search.startedAt)}</Data>
    </span>
  );
  // A finished run's time sits in its meta line, so a run's opened Releases
  // take the row's full width.
  return (
    <ListRow
      title={search.outcome === null ? "Searching" : outcomeText[search.outcome]}
      meta={
        search.outcome === null ? undefined : (
          <>
            {found(search.candidates)} · {time}
          </>
        )
      }
      detail={
        search.indexers.length === 0 && !releases ? undefined : (
          <RunDetails search={search} releases={releases} />
        )
      }
      trailing={
        search.outcome === null ? (
          <span className="flex items-center gap-3">
            <Spinner />
            {time}
          </span>
        ) : undefined
      }
    />
  );
}

export function Searches({ workId }: { workId: string }) {
  const { searches } = useListSearchesSuspense(workId).data;
  const latest = latestRuns(searches);
  return (
    <Section title="Searches" count={searches.length}>
      {searches.length === 0 ? (
        <p className="text-ink-3">No searches yet. Search looks for a Release on every Indexer.</p>
      ) : (
        <List label="Searches">
          {searches.map((search) => (
            <SearchRow
              key={search.id}
              search={search}
              releases={latest.has(search.id) && search.candidates > 0}
            />
          ))}
        </List>
      )}
    </Section>
  );
}
