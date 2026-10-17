import type { WorksContentType, WorksTree } from "../../../api/gen/model";
import { useGetWorkCellsSuspense } from "../../../api/gen/query/files/files";
import { useListContentTypesSuspense } from "../../../api/gen/query/works/works";
import { SearchAction } from "../../../grabs/search";
import { Grabs, NeedsYou, Searches } from "../../../grabs/work-sections";
import { Page } from "../../../shell/page";
import { fieldText, typeOf } from "../../../works/fields";
import { Files } from "../../../works/files";
import { Targets } from "../../../works/targets";

export function WorkBody({ workId, work }: { workId: string; work: WorksTree }) {
  const cells = useGetWorkCellsSuspense(workId).data;
  const type = typeOf(useListContentTypesSuspense().data.contentTypes, work.contentType);
  return (
    <Page title={work.label} action={<SearchAction workId={workId} />}>
      <WorkFields type={type} work={work} />
      <NeedsYou workId={workId} work={work.label} />
      <Targets type={type} work={work} cells={cells} />
      <Grabs workId={workId} work={work.label} />
      <Searches workId={workId} />
      <Files cells={cells.cells} />
    </Page>
  );
}

function WorkFields({ type, work }: { type: WorksContentType; work: WorksTree }) {
  const labels = new Map(type.fields.map((spec) => [spec.key, spec.label]));
  return (
    <dl className="mb-6 flex flex-wrap gap-x-8 gap-y-3">
      {work.fields.map((field) => (
        <div key={field.key} className="flex flex-col gap-0.5">
          <dt className="text-label text-ink-3">{labels.get(field.key)}</dt>
          <dd className="text-ink-1 tabular-nums">{fieldText(field)}</dd>
        </div>
      ))}
    </dl>
  );
}
