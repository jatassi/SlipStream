import { Link } from "@tanstack/react-router";

import type { WorksCandidate, WorksContentType } from "../../api/gen/model";
import { useSearchCatalogSuspense } from "../../api/gen/query/works/works";
import { Empty } from "../../shell/empty";
import { Button } from "../../ui/button";
import { Chip } from "../../ui/chip";
import { List, ListRow } from "../../ui/list";
import { Poster } from "../../ui/poster";
import { factTexts } from "../page/facts";
import { openAddSheet } from "./sheet-state";

const shown = 25;

export function Candidates({ type, query }: { type: WorksContentType; query: string }) {
  const candidates = useSearchCatalogSuspense(type.id, { query }).data.candidates.slice(0, shown);
  if (candidates.length === 0) {
    return (
      <Empty>
        Nothing matches &quot;{query}&quot;. Try another spelling, or the {type.work.one}&apos;s
        original title.
      </Empty>
    );
  }
  return (
    <List label="Results">
      {candidates.map((candidate) => (
        <Candidate key={candidate.key} type={type} candidate={candidate} />
      ))}
    </List>
  );
}

function Meta({ type, candidate }: { type: WorksContentType; candidate: WorksCandidate }) {
  const facts = [...factTexts(type, candidate.fields), candidate.original].filter(
    (text) => text !== "",
  );
  return (
    <>
      <span className="block truncate">{facts.join(" · ")}</span>
      {candidate.overview !== "" && (
        <span className="block truncate text-ink-2">{candidate.overview}</span>
      )}
    </>
  );
}

function Candidate({ type, candidate }: { type: WorksContentType; candidate: WorksCandidate }) {
  const poster = <Poster url={candidate.posterUrl ?? undefined} />;
  if (candidate.workId !== null) {
    return (
      <ListRow
        leading={poster}
        title={candidate.label}
        meta={<Meta type={type} candidate={candidate} />}
        trailing={<Chip>In the library</Chip>}
        render={<Link to="/works/$workId" params={{ workId: candidate.workId }} />}
      />
    );
  }
  return (
    <ListRow
      leading={poster}
      title={candidate.label}
      meta={<Meta type={type} candidate={candidate} />}
      trailing={
        <Button
          aria-label={`Add ${candidate.label}`}
          size="small"
          onClick={() => {
            openAddSheet({ type, candidate });
          }}
        >
          Add
        </Button>
      }
    />
  );
}
