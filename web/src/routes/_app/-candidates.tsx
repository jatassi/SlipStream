import { Link, useNavigate } from "@tanstack/react-router";

import type { WorksCandidate, WorksContentType } from "../../api/gen/model";
import { useAddWork, useSearchCatalogSuspense } from "../../api/gen/query/works/works";
import { Button } from "../../ui/button";
import { submitErrors } from "../../ui/form";
import { fieldText } from "../../works/fields";

export function Candidates({ type, query }: { type: WorksContentType; query: string }) {
  const { candidates } = useSearchCatalogSuspense(type.id, { query }).data;
  if (candidates.length === 0) {
    return <p className="text-ink-3">Nothing matches “{query}”.</p>;
  }
  return (
    <ul aria-label="Results" className="flex flex-col gap-3">
      {candidates.map((candidate) => (
        <li key={candidate.key} className="flex flex-wrap items-center gap-4">
          <span>
            {candidate.label}
            <span className="text-ink-3"> {candidate.fields.map(fieldText).join(" · ")}</span>
          </span>
          <CandidateAction type={type} candidate={candidate} />
        </li>
      ))}
    </ul>
  );
}

function CandidateAction({
  type,
  candidate,
}: {
  type: WorksContentType;
  candidate: WorksCandidate;
}) {
  const navigate = useNavigate();
  const addWork = useAddWork({
    mutation: {
      onSuccess: async ({ work }) => {
        await navigate({ to: "/works/$workId", params: { workId: work.id } });
      },
    },
  });
  if (candidate.workId !== null) {
    return (
      <Link
        to="/works/$workId"
        params={{ workId: candidate.workId }}
        aria-label={`Open ${candidate.label}`}
        className="underline"
      >
        Open
      </Link>
    );
  }
  return (
    <>
      <Button
        aria-label={`Add ${candidate.label}`}
        disabled={addWork.isPending}
        onClick={() => {
          addWork.mutate({ data: { contentType: type.id, key: candidate.key, monitor: true } });
        }}
      >
        Add
      </Button>
      {addWork.isError ? (
        <p role="alert" className="text-fail-ink">
          {submitErrors(addWork.error).form}
        </p>
      ) : null}
    </>
  );
}
