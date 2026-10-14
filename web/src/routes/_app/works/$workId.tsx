import { createFileRoute } from "@tanstack/react-router";

import type { FilesCell, FilesMetCell, WorksContentType, WorksTree } from "../../../api/gen/model";
import {
  getGetWorkCellsSuspenseQueryOptions,
  useGetWorkCellsSuspense,
} from "../../../api/gen/query/files/files";
import {
  getListGrabsSuspenseQueryOptions,
  getListSearchesSuspenseQueryOptions,
} from "../../../api/gen/query/grabs/grabs";
import {
  getGetWorkSuspenseQueryOptions,
  getListContentTypesSuspenseQueryOptions,
  useGetWorkSuspense,
  useListContentTypesSuspense,
} from "../../../api/gen/query/works/works";
import { ensureQuery } from "../../../api/query-client";
import { Grabs, SearchButton, Searches } from "../../../grabs/work-sections";
import { Skeleton } from "../../../ui/skeleton";
import { fieldText, typeOf } from "../../../works/fields";

export const Route = createFileRoute("/_app/works/$workId")({
  loader: ({ context, params: { workId } }) =>
    Promise.all([
      ensureQuery(context.queryClient, getListContentTypesSuspenseQueryOptions()),
      ensureQuery(context.queryClient, getGetWorkSuspenseQueryOptions(workId)),
      ensureQuery(context.queryClient, getGetWorkCellsSuspenseQueryOptions(workId)),
      ensureQuery(context.queryClient, getListSearchesSuspenseQueryOptions(workId)),
      ensureQuery(context.queryClient, getListGrabsSuspenseQueryOptions()),
    ]),
  pendingComponent: () => <Skeleton label="Loading the Work" />,
  component: WorkPage,
});

function WorkPage() {
  const { workId } = Route.useParams();
  const work = useGetWorkSuspense(workId).data;
  const { cells } = useGetWorkCellsSuspense(workId).data;
  const type = typeOf(useListContentTypesSuspense().data.contentTypes, work.contentType);
  return (
    <article className="flex flex-col gap-8">
      <WorkHeader type={type} work={work} />
      <SearchButton workId={workId} />
      <CellsGrid type={type} work={work} cells={cells} />
      <Files cells={cells} />
      <Searches workId={workId} />
      <Grabs workId={workId} />
    </article>
  );
}

function WorkHeader({ type, work }: { type: WorksContentType; work: WorksTree }) {
  const labels = new Map(type.fields.map((spec) => [spec.key, spec.label]));
  return (
    <header className="flex flex-col gap-4">
      <h1 className="text-2xl font-bold">{work.label}</h1>
      <dl className="grid grid-cols-[max-content_1fr] gap-x-6 gap-y-2">
        {work.fields.map((field) => (
          <div key={field.key} className="contents">
            <dt className="text-ink-3">{labels.get(field.key)}</dt>
            <dd>{fieldText(field)}</dd>
          </div>
        ))}
      </dl>
    </header>
  );
}

const fileName = (path: string) =>
  path.slice(Math.max(path.lastIndexOf("/"), path.lastIndexOf("\\")) + 1);

function CellsGrid({
  type,
  work,
  cells,
}: {
  type: WorksContentType;
  work: WorksTree;
  cells: FilesCell[];
}) {
  const targets = [...new Map(cells.map((cell) => [cell.targetId, cell.targetName]))];
  const cellAt = new Map(cells.map((cell) => [`${cell.itemId} ${cell.targetId}`, cell]));
  return (
    <section className="flex flex-col gap-3">
      <h2 className="text-xl font-bold">Cells</h2>
      <table className="text-left">
        <thead>
          <tr>
            <th scope="col" className="pr-6 font-semibold">
              {type.item.one}
            </th>
            {targets.map(([id, name]) => (
              <th key={id} scope="col" className="pr-6 font-semibold">
                {name}
              </th>
            ))}
          </tr>
        </thead>
        <tbody>
          {work.items.map((item) => (
            <tr key={item.id}>
              <th scope="row" className="pr-6 font-normal">
                {item.label}
              </th>
              {targets.map(([id]) => (
                <td key={id} className="pr-6">
                  <CellState cell={cellAt.get(`${item.id} ${id}`)} />
                </td>
              ))}
            </tr>
          ))}
        </tbody>
      </table>
    </section>
  );
}

const stateText: Record<FilesCell["state"], string> = {
  missing: "Missing",
  downloading: "Downloading",
  failed: "Failed",
  met: "Met",
};

function CellState({ cell }: { cell: FilesCell | undefined }) {
  return (
    <>
      {cell === undefined ? null : stateText[cell.state]}
      {cell?.state === "met" ? (
        <span className="block font-mono text-sm wrap-anywhere">{fileName(cell.file.path)}</span>
      ) : null}
    </>
  );
}

function Files({ cells }: { cells: FilesCell[] }) {
  const files = [
    ...new Map(
      cells
        .filter((cell): cell is FilesMetCell => cell.state === "met")
        .map((cell) => [cell.file.id, cell.file]),
    ).values(),
  ];
  return (
    <section className="flex flex-col gap-3">
      <h2 className="text-xl font-bold">Files</h2>
      {files.length === 0 ? (
        <p className="text-ink-3">No files yet.</p>
      ) : (
        <ul aria-label="Files" className="flex flex-col gap-2">
          {files.map((file) => (
            <li key={file.id}>
              <span className="font-mono wrap-anywhere">{file.path}</span>
              <span className="text-ink-3">
                {" "}
                {file.probe.width}×{file.probe.height}
              </span>
            </li>
          ))}
        </ul>
      )}
    </section>
  );
}
