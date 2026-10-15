import { Link, useNavigate } from "@tanstack/react-router";

import type { WorksCandidate, WorksContentType } from "../../api/gen/model";
import { useAddWork, useSearchCatalogSuspense } from "../../api/gen/query/works/works";
import { Empty } from "../../shell/empty";
import { Button } from "../../ui/button";
import { Chip } from "../../ui/chip";
import { List, ListRow } from "../../ui/list";
import { toast } from "../../ui/toast";
import { fieldText } from "../../works/fields";

export function Candidates({ type, query }: { type: WorksContentType; query: string }) {
  const { candidates } = useSearchCatalogSuspense(type.id, { query }).data;
  if (candidates.length === 0) {
    return (
      <Empty>
        Nothing matches “{query}”. Try another spelling, or the {type.work.one}&apos;s original
        title.
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
      trailing={<AddCandidate type={type} candidate={candidate} />}
    />
  );
}

function AddCandidate({ type, candidate }: { type: WorksContentType; candidate: WorksCandidate }) {
  const navigate = useNavigate();
  const addWork = useAddWork({
    mutation: {
      onSuccess: async ({ created, work }) => {
        toast(created ? `Added ${work.label}` : `${work.label} is already in the library`);
        await navigate({ to: "/works/$workId", params: { workId: work.id } });
      },
      onError: (error) => {
        toast(error.message, { kind: "fail" });
      },
    },
  });
  return (
    <Button
      aria-label={`Add ${candidate.label}`}
      size="small"
      busy={addWork.isPending}
      onClick={() => {
        addWork.mutate({ data: { contentType: type.id, key: candidate.key, monitor: true } });
      }}
    >
      Add
    </Button>
  );
}
