import type { FilesCell, FilesLibraryFile } from "../api/gen/model";
import { bytes } from "../format";
import { Data } from "../ui/data";
import { List, ListRow } from "../ui/list";
import { Section } from "../ui/section";
import { folderOf, nameOf } from "./paths";

function FileRow({ file }: { file: FilesLibraryFile }) {
  return (
    <ListRow
      title={<Data>{nameOf(file.path)}</Data>}
      meta={<Data>{folderOf(file.path)}</Data>}
      trailing={
        <span className="text-meta">
          <Data>
            {file.probe.width}×{file.probe.height}
          </Data>{" "}
          · <Data>{bytes(file.size)}</Data>
        </span>
      }
    />
  );
}

export function Files({ cells }: { cells: readonly FilesCell[] }) {
  const files = [
    ...new Map(
      cells.flatMap((cell) => ("file" in cell ? [[cell.file.id, cell.file] as const] : [])),
    ).values(),
  ];
  return (
    <Section title="Files" count={files.length}>
      {files.length === 0 ? (
        <p className="text-ink-3">No files yet. A Grab that imports puts its file here.</p>
      ) : (
        <List label="Files">
          {files.map((file) => (
            <FileRow key={file.id} file={file} />
          ))}
        </List>
      )}
    </Section>
  );
}
