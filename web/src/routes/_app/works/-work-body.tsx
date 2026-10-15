import type { FilesCell, FilesMetCell, WorksContentType, WorksTree } from "../../../api/gen/model";
import { useGetWorkCellsSuspense } from "../../../api/gen/query/files/files";
import { useListContentTypesSuspense } from "../../../api/gen/query/works/works";
import { Grabs, SearchButton, Searches } from "../../../grabs/work-sections";
import { Data } from "../../../ui/data";
import { fieldText, typeOf } from "../../../works/fields";

export function WorkBody({ workId, work }: { workId: string; work: WorksTree }) {
  const { cells } = useGetWorkCellsSuspense(workId).data;
  const type = typeOf(useListContentTypesSuspense().data.contentTypes, work.contentType);
  return (
    <article className="flex flex-col gap-8">
      <WorkFields type={type} work={work} />
      <SearchButton workId={workId} />
      <CellsGrid type={type} work={work} cells={cells} />
      <Files cells={cells} />
      <Searches workId={workId} />
      <Grabs workId={workId} />
    </article>
  );
}

function WorkFields({ type, work }: { type: WorksContentType; work: WorksTree }) {
  const labels = new Map(type.fields.map((spec) => [spec.key, spec.label]));
  return (
    <dl className="grid grid-cols-[max-content_1fr] gap-x-6 gap-y-2">
      {work.fields.map((field) => (
        <div key={field.key} className="contents">
          <dt className="text-ink-3">{labels.get(field.key)}</dt>
          <dd>{fieldText(field)}</dd>
        </div>
      ))}
    </dl>
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
        <Data className="block wrap-anywhere">{fileName(cell.file.path)}</Data>
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
              <Data className="wrap-anywhere">{file.path}</Data>
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
