import type { GrabsSearch } from "../api/gen/model";
import {
  useListGrabsSuspense,
  useListSearchesSuspense,
  useSearchWork,
} from "../api/gen/query/grabs/grabs";
import { Button } from "../ui/button";
import { submitErrors } from "../ui/form";
import { GrabStatus } from "./grab-status";
import { Data } from "../ui/data";

export function SearchButton({ workId }: { workId: string }) {
  const search = useSearchWork();
  return (
    <div className="flex flex-wrap items-center gap-4">
      <Button
        variant="primary"
        disabled={search.isPending}
        onClick={() => {
          search.mutate({ workId });
        }}
      >
        Search
      </Button>
      {search.isError ? (
        <p role="alert" className="text-fail-ink">
          {submitErrors(search.error).form}
        </p>
      ) : null}
    </div>
  );
}

const outcomeText: Record<NonNullable<GrabsSearch["outcome"]>, string> = {
  grabbed: "Grabbed",
  nothing_acceptable: "Nothing acceptable",
  indexer_unavailable: "Indexer unavailable",
  indexer_refused: "Indexer refused",
};

const releases = (count: number) => (count === 1 ? "1 release" : `${count} releases`);

export function Searches({ workId }: { workId: string }) {
  const { searches } = useListSearchesSuspense(workId).data;
  return (
    <section className="flex flex-col gap-3">
      <h2 className="text-xl font-bold">Searches</h2>
      {searches.length === 0 ? (
        <p className="text-ink-3">No searches yet.</p>
      ) : (
        <ul aria-label="Searches" className="flex flex-col gap-2">
          {searches.map((search) => (
            <li key={search.id}>
              {search.outcome === null
                ? "Searching"
                : `${outcomeText[search.outcome]} · ${releases(search.candidates)}`}
            </li>
          ))}
        </ul>
      )}
    </section>
  );
}

export function Grabs({ workId }: { workId: string }) {
  const grabs = useListGrabsSuspense().data.grabs.filter((grab) => grab.workId === workId);
  return (
    <section className="flex flex-col gap-3">
      <h2 className="text-xl font-bold">Grabs</h2>
      {grabs.length === 0 ? (
        <p className="text-ink-3">No Grabs yet.</p>
      ) : (
        <ul aria-label="Grabs" className="flex flex-col gap-3">
          {grabs.map((grab) => (
            <li key={grab.id} className="flex flex-col gap-1">
              <Data className="wrap-anywhere">{grab.releaseTitle}</Data>
              <GrabStatus grab={grab} />
            </li>
          ))}
        </ul>
      )}
    </section>
  );
}
