import { sentenceOf } from "../api/errors";
import type { WorksCandidate, WorksContentType } from "../api/gen/model";
import type { RootEntry } from "./root-entry";
import { useActOnRootImport } from "../api/gen/query/imports/imports";
import { useSearchCatalogSuspense } from "../api/gen/query/works/works";
import { Button } from "../ui/button";
import { List, ListRow } from "../ui/list";
import { Poster } from "../ui/poster";
import { toast } from "../ui/toast";
import { named } from "./words";

const shown = 25;

const yearOf = (candidate: WorksCandidate) => {
  const year = candidate.fields.find(({ key }) => key === "year");
  return year?.kind === "number" ? year.number : null;
};

function CandidateRow({
  entry,
  candidate,
  onDone,
}: {
  entry: RootEntry;
  candidate: WorksCandidate;
  onDone: () => void;
}) {
  const match = useActOnRootImport();
  const label = named(candidate.label, yearOf(candidate));
  return (
    <ListRow
      leading={<Poster url={candidate.posterUrl ?? undefined} />}
      title={label}
      meta={candidate.overview === "" ? undefined : candidate.overview}
      trailing={
        <Button
          size="small"
          aria-label={`Match ${label}`}
          disabled={candidate.unlisted}
          busy={match.isPending}
          onClick={() => {
            match.mutate(
              { id: entry.id, data: { action: "match", key: candidate.key } },
              {
                onSuccess: () => {
                  toast(`${entry.folder} now matches ${label}.`);
                  onDone();
                },
                onError: (error) => {
                  toast(sentenceOf(error), { kind: "fail" });
                },
              },
            );
          }}
        >
          Match
        </Button>
      }
    />
  );
}

/** The source's records for a query, each with Match, which makes it the entry's suggestion. */
export function MatchResults({
  entry,
  type,
  query,
  onDone,
}: {
  entry: RootEntry;
  type: WorksContentType;
  query: string;
  onDone: () => void;
}) {
  const candidates = useSearchCatalogSuspense(type.id, { query }).data.candidates.slice(0, shown);
  if (candidates.length === 0) {
    return (
      <p className="text-ink-2">
        Nothing matches &quot;{query}&quot;. Try another spelling, or the{" "}
        {type.work.one.toLowerCase()}&apos;s original title.
      </p>
    );
  }
  return (
    <List label="Results">
      {candidates.map((candidate) => (
        <CandidateRow key={candidate.key} entry={entry} candidate={candidate} onDone={onDone} />
      ))}
    </List>
  );
}
