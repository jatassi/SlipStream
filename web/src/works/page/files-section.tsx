import type { FilesFileList, FilesListedFile, WorksWorkRoot } from "../../api/gen/model";
import { bytes } from "../../format";
import { useQualityNames } from "../../targets/quality-names";
import { Chip } from "../../ui/chip";
import { Data } from "../../ui/data";
import { List, ListRow } from "../../ui/list";
import { Section } from "../../ui/section";
import { nameOf } from "../paths";
import { anyCut } from "./matrix-model";

type Names = {
  readonly cutName: (key: string) => string;
  readonly targetName: (id: string) => string;
};

const fillWords =
  ({ cutName, targetName }: Names) =>
  (file: FilesListedFile) =>
    `Fills ${file.fills
      .map(({ targetId, edition }) =>
        edition === anyCut ? targetName(targetId) : `${targetName(targetId)}, ${cutName(edition)}`,
      )
      .join("; ")}`;

function FileRow({
  file,
  fills,
}: {
  file: FilesListedFile;
  fills: (file: FilesListedFile) => string;
}) {
  const quality = useQualityNames();
  return (
    <ListRow
      title={<Data>{nameOf(file.path)}</Data>}
      meta={
        <>
          <Data>{file.folder}</Data> · {fills(file)}
        </>
      }
      trailing={
        <span className="flex items-center gap-2 text-meta">
          {file.kept && <Chip look="outline">Kept file</Chip>}
          <Data>
            {quality(file.quality)} · {file.probe.width}×{file.probe.height} · {bytes(file.size)}
          </Data>
        </span>
      }
    />
  );
}

/** Every library file of the Work, the ones that fill its cells first, then Kept files apart. */
export function FilesSection({
  listing: { folder, files },
  root,
  names,
}: {
  listing: FilesFileList;
  root: WorksWorkRoot | null;
  names: Names;
}) {
  const filling = files.filter((file) => !file.kept);
  const kept = files.filter((file) => file.kept);
  return (
    <Section title="Files" count={files.length}>
      <dl className="mb-3 flex flex-wrap gap-x-8 gap-y-2">
        <div className="flex min-w-0 flex-col gap-0.5">
          <dt className="text-label text-ink-3">Library root</dt>
          <dd className="text-ink-1">{root === null ? "None yet" : <Data>{root.path}</Data>}</dd>
        </div>
        <div className="flex min-w-0 flex-col gap-0.5">
          <dt className="text-label text-ink-3">Folder</dt>
          <dd className="text-ink-1">
            <Data>{folder}</Data>
          </dd>
        </div>
      </dl>
      {files.length === 0 ? (
        <p className="text-ink-3">No files yet. A Grab that imports puts its file here.</p>
      ) : (
        <div className="flex flex-col gap-3">
          {filling.length > 0 && (
            <List label="Files">
              {filling.map((file) => (
                <FileRow key={file.id} file={file} fills={fillWords(names)} />
              ))}
            </List>
          )}
          {kept.length > 0 && (
            <List label="Kept files">
              {kept.map((file) => (
                <FileRow key={file.id} file={file} fills={() => "Fills no wanted Target"} />
              ))}
            </List>
          )}
        </div>
      )}
    </Section>
  );
}
