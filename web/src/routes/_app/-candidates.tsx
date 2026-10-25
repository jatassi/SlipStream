import { Link } from "@tanstack/react-router";

import type { WorksCandidate, WorksContentType } from "../../api/gen/model";
import { useSearchCatalogSuspense } from "../../api/gen/query/works/works";
import { Empty } from "../../shell/empty";
import { Chip } from "../../ui/chip";
import { List, ListRow } from "../../ui/list";
import { AddSheet } from "../../works/add-sheet";
import { fieldText } from "../../works/fields";

export function Candidates({ type, query }: { type: WorksContentType; query: string }) {
  const { candidates } = useSearchCatalogSuspense(type.id, { query }).data;
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

function Candidate({ type, candidate }: { type: WorksContentType; candidate: WorksCandidate }) {
  const meta = candidate.fields.map(fieldText).join(" · ");
  if (candidate.workId !== null) {
    return (
      <ListRow
        title={candidate.label}
        meta={meta}
        trailing={<Chip>In the library</Chip>}
        render={<Link to="/works/$workId" params={{ workId: candidate.workId }} />}
      />
    );
  }
  return (
    <ListRow
      title={candidate.label}
      meta={meta}
      trailing={<AddSheet type={type} candidate={candidate} />}
    />
  );
}
