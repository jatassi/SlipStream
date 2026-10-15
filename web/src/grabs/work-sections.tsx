import type { GrabsSearch } from "../api/gen/model";
import { useListGrabsSuspense, useListSearchesSuspense } from "../api/gen/query/grabs/grabs";
import { clock } from "../format";
import { Data } from "../ui/data";
import { List, ListRow } from "../ui/list";
import { Section } from "../ui/section";
import { Spinner } from "../ui/spinner";
import { GrabFailure } from "./failure-cards";
import { GrabTable } from "./grab-table";
import { isFailed } from "./stages";

const outcomeText: Record<NonNullable<GrabsSearch["outcome"]>, string> = {
  grabbed: "Grabbed a release",
  nothing_acceptable: "Found nothing acceptable",
  indexer_unavailable: "The Indexer didn't answer",
  indexer_refused: "The Indexer refused the search",
};

const found = (count: number) => (count === 1 ? "1 release found" : `${count} releases found`);

function SearchRow({ search }: { search: GrabsSearch }) {
  const time = <Data>{clock(search.startedAt)}</Data>;
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
        search.outcome === "grabbed" || search.outcome === "nothing_acceptable"
          ? found(search.candidates)
          : undefined
      }
      trailing={time}
    />
  );
}

export function Searches({ workId }: { workId: string }) {
  const { searches } = useListSearchesSuspense(workId).data;
  return (
    <Section title="Searches" count={searches.length}>
      {searches.length === 0 ? (
        <p className="text-ink-3">No searches yet. Search looks for a release on every Indexer.</p>
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

function useWorkGrabs(workId: string) {
  return useListGrabsSuspense().data.grabs.filter((grab) => grab.workId === workId);
}

export function NeedsYou({ workId, work }: { workId: string; work: string }) {
  const failed = useWorkGrabs(workId).filter((grab) => isFailed(grab));
  if (failed.length === 0) {
    return null;
  }
  return (
    <Section title="Needs you" count={failed.length}>
      <div className="flex flex-col gap-3">
        {failed.map((grab) => (
          <GrabFailure key={grab.id} grab={grab} work={work} form="compact" />
        ))}
      </div>
    </Section>
  );
}

export function Grabs({ workId, work }: { workId: string; work: string }) {
  const grabs = useWorkGrabs(workId).filter((grab) => !isFailed(grab));
  return (
    <Section title="Grabs" count={grabs.length}>
      {grabs.length === 0 ? (
        <p className="text-ink-3">No Grabs yet. A search that finds a release grabs it.</p>
      ) : (
        <GrabTable layout="work" rows={grabs.map((grab) => ({ grab, work }))} />
      )}
    </Section>
  );
}
