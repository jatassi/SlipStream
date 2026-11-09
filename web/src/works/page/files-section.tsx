import type { FilesFileList, FilesListedFile, WorksWorkRoot } from "../../api/gen/model";
import { bytes } from "../../format";
import { RenamingChip } from "../../renames/renaming-chip";
import { useQualityNames } from "../../targets/quality-names";
import { Chip } from "../../ui/chip";
import { Data } from "../../ui/data";
import { List, ListRow } from "../../ui/list";
import { Section } from "../../ui/section";
import { nameOf } from "../paths";
import { FileMedia, ForgetFile, goneWords } from "./file-media";
// oxlint-disable-next-line import/max-dependencies -- the section lists the files with their media, qualities and the renames still moving them
import { anyCut } from "./matrix-model";

type Names = {
  readonly cutName: (key: string) => string;
  readonly targetName: (id: string) => string;
};

const list = new Intl.ListFormat("en", { type: "conjunction" });

// "Fills Director's Cut 2160p and 1080p", a cut before its Target as Needs you says it.
const fillWords =
  ({ cutName, targetName }: Names) =>
  (file: FilesListedFile) =>
    `Fills ${list.format(
      file.fills.map(({ targetId, edition }) =>
        edition === anyCut ? targetName(targetId) : `${cutName(edition)} ${targetName(targetId)}`,
      ),
    )}`;

function FileRow({ file, about }: { file: FilesListedFile; about: string }) {
  const quality = useQualityNames();
  const gone = file.goneAt !== null;
  return (
    <ListRow
      title={<Data>{nameOf(file.path)}</Data>}
      meta={
        <>
          <Data>{file.folder}</Data> · {about}
        </>
      }
      detail={<FileMedia file={file} />}
      trailing={
        <span className="flex items-center gap-2 text-meta">
          {file.kept && <Chip look="outline">Kept file</Chip>}
          <Data>
            {quality(file.quality)} ·{" "}
            {!gone && file.media && `${file.media.video.width}×${file.media.video.height} · `}
            {bytes(file.size)}
          </Data>
          {gone && <ForgetFile file={file} />}
        </span>
      }
    />
  );
}

type Group = "filling" | "kept" | "gone";

const groups: readonly { readonly group: Group; readonly label: string }[] = [
  { group: "filling", label: "Files" },
  { group: "kept", label: "Kept files" },
  { group: "gone", label: "Gone files" },
];

function placed(file: FilesListedFile, fills: (file: FilesListedFile) => string) {
  if (file.goneAt !== null) {
    return { file, group: "gone", about: goneWords(file.goneAt) } as const;
  }
  return file.kept
    ? ({ file, group: "kept", about: "Fills no wanted Target" } as const)
    : ({ file, group: "filling", about: fills(file) } as const);
}

function FileLists({ files, names }: { files: readonly FilesListedFile[]; names: Names }) {
  const fills = fillWords(names);
  const rows = files.map((file) => placed(file, fills));
  return (
    <div className="flex flex-col gap-3">
      {groups.map(({ group, label }) => {
        const listed = rows.filter((row) => row.group === group);
        return (
          listed.length > 0 && (
            <List key={group} label={label}>
              {listed.map(({ file, about }) => (
                <FileRow key={file.id} file={file} about={about} />
              ))}
            </List>
          )
        );
      })}
    </div>
  );
}

/** Every library file of the Work, the ones that fill its cells first, then Kept files and gone files apart. */
export function FilesSection({
  listing: { folder, files },
  root,
  names,
  workId,
}: {
  listing: FilesFileList;
  root: WorksWorkRoot | null;
  names: Names;
  workId: string;
}) {
  return (
    <Section title="Files" count={files.length} chip={<RenamingChip params={{ workId }} />}>
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
        <FileLists files={files} names={names} />
      )}
    </Section>
  );
}
