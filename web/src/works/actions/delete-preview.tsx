import { keepPreviousData, useQuery } from "@tanstack/react-query";

import { sentenceOf } from "../../api/errors";
import type {
  WorksDeletingFiles,
  WorksFailedOutcome,
  WorksOutcomeProblem,
} from "../../api/gen/model";
import { deleteWorks } from "../../api/gen/query/works/works";
import { Button } from "../../ui/button";
import { Spinner } from "../../ui/spinner";
import type { Named, Subject } from "./actions";
import { planWords } from "./delete-plan";

/** A dry run of the same delete: its plan, and the Works it would refuse. */
export function useDeletePreview(subjects: readonly Subject[], files: WorksDeletingFiles) {
  const workIds = subjects.map(({ id }) => id);
  return useQuery({
    queryKey: ["works-delete-preview", files, ...workIds],
    queryFn: ({ signal }) => deleteWorks({ workIds, files, preview: true }, { signal }),
    gcTime: 0,
    placeholderData: keepPreviousData,
  });
}

type Preview = ReturnType<typeof useDeletePreview>;

const offlineOf = (preview: Preview): ReadonlyMap<string, WorksOutcomeProblem> =>
  new Map(
    (preview.data?.outcomes ?? [])
      .filter(
        (outcome): outcome is WorksFailedOutcome =>
          outcome.outcome === "failed" && outcome.problem.code === "storage_unreachable",
      )
      .map(({ workId, problem }) => [workId, problem]),
  );

function Offline({
  subjects,
  preview,
  onLeave,
}: {
  subjects: readonly Subject[];
  preview: Preview;
  onLeave: () => void;
}) {
  const offline = offlineOf(preview);
  const stuck = subjects.flatMap((subject) => {
    const problem = offline.get(subject.id);
    return problem === undefined ? [] : [{ subject, problem }];
  });
  if (stuck.length === 0) {
    return null;
  }
  return (
    <div className="flex flex-col items-start gap-2">
      {stuck.map(({ subject, problem }) => (
        <p key={subject.id} className="text-fail-ink">
          {`${subject.label} can't be deleted with its files. ${problem.detail ?? `${problem.title}.`}`}
        </p>
      ))}
      <Button onClick={onLeave}>Leave files</Button>
    </div>
  );
}

/** The plan in words, and each Work whose files sit on an offline Node with a way to leave them. */
export function DeletePreview({
  subjects,
  named,
  files,
  preview,
  onLeave,
}: {
  subjects: readonly Subject[];
  named: Named;
  files: WorksDeletingFiles;
  preview: Preview;
  onLeave: () => void;
}) {
  if (preview.isPending || preview.isPlaceholderData) {
    return <Spinner label="Checking what this deletes" />;
  }
  if (preview.isError) {
    return <p className="text-fail-ink">{sentenceOf(preview.error)}</p>;
  }
  return (
    <>
      {planWords(preview.data.plan, files, named).map((line) => (
        <p key={line}>{line}</p>
      ))}
      <Offline subjects={subjects} preview={preview} onLeave={onLeave} />
    </>
  );
}
