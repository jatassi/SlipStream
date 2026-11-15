import type { GrabsGrab, GrabsSearch } from "../api/gen/model";
import { useListGrabsSuspense, useListSearchesSuspense } from "../api/gen/query/grabs/grabs";
import { clock } from "../format";
import { useTester } from "../formats/tester-dialog";
import type { WorkRef } from "../formats/testing";
import { Data } from "../ui/data";
import { List, ListRow } from "../ui/list";
import { Section } from "../ui/section";
import { Spinner } from "../ui/spinner";
import { RunReleases } from "./search-releases";
// oxlint-disable-next-line import/max-dependencies -- a grabbed run reads its Grab's stage
import { outcomeWords, triggerWords } from "./search-words";
import { stageWord } from "./stages";

const found = (count: number) => (count === 1 ? "1 Release found" : `${count} Releases found`);

/** What started the run, and its group: "Backlog search · season 3". */
const startedBy = ({ trigger, groupLabel }: GrabsSearch): string =>
  [trigger === null ? "Manual search" : triggerWords[trigger], groupLabel]
    .filter((part) => part !== null)
    .join(" · ");

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

type Run = {
  search: GrabsSearch;
  grab: GrabsGrab | undefined;
  releases: boolean;
  onTest: (title: string) => void;
};

function RunDetails({ search, releases, onTest }: Omit<Run, "grab">) {
  return (
    <>
      {search.indexers.length > 0 && <IndexerLines search={search} />}
      {releases && <RunReleases search={search} onTest={onTest} />}
    </>
  );
}

function SearchRow({ search, grab, releases, onTest }: Run) {
  const time = (
    <span className="text-meta">
      <Data>{clock(search.startedAt)}</Data>
    </span>
  );
  const stage = grab !== undefined && (
    <>
      <span className={grab.stage === "failed" ? "text-fail-ink" : undefined}>
        {stageWord[grab.stage]}
      </span>
      {" · "}
    </>
  );
  // A finished run's time sits in its meta line, so a run's opened Releases
  // take the row's full width.
  return (
    <ListRow
      title={search.outcome === null ? "Searching" : outcomeWords[search.outcome](search)}
      meta={
        <>
          {startedBy(search)}
          {search.outcome !== null && (
            <>
              {" · "}
              {found(search.candidates)} · {stage}
              {time}
            </>
          )}
        </>
      }
      detail={
        search.indexers.length === 0 && !releases ? undefined : (
          <RunDetails search={search} releases={releases} onTest={onTest} />
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

/** The Work's search runs; a Release of one opens the tester over this page, judged against the Work. */
export function Searches({ work }: { work: WorkRef }) {
  const { searches } = useListSearchesSuspense(work.id).data;
  const { grabs } = useListGrabsSuspense().data;
  const tester = useTester();
  const onTest = (title: string) => {
    tester.open({ names: [title], work });
  };
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
              grab={grabs.find(({ id }) => id === search.grabIds[0])}
              releases={search.outcome !== null && search.candidates > 0}
              onTest={onTest}
            />
          ))}
        </List>
      )}
      {tester.dialog}
    </Section>
  );
}
