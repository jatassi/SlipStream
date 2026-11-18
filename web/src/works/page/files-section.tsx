import type {
  FilesFileList,
  FilesListedFile,
  SubtitlesFileSubtitles,
  SubtitlesProvider,
  WorksWorkRoot,
} from "../../api/gen/model";
import { bytes } from "../../format";
import { RenamingChip } from "../../renames/renaming-chip";
import { useWorkSubtitles } from "../../subtitles/work-api";
import { rowStateOf } from "../../subtitles/work-words";
import { useQualityNames } from "../../targets/quality-names";
import { Chip } from "../../ui/chip";
import { Data } from "../../ui/data";
import { List, ListRow } from "../../ui/list";
import { Section } from "../../ui/section";
// oxlint-disable-next-line import/max-dependencies -- the section lists the files with their media, qualities, subtitles and the renames still moving them
import { nameOf } from "../paths";
import { FileMedia, ForgetFile, goneWords } from "./file-media";
import { anyCut } from "./matrix-model";
import { type Season, SeasonFolds } from "./season-fold";
import { type SubtitleFile, SubtitleDetails } from "./subtitle-details";
import { SubtitlesBlock } from "./subtitles-block";

type Names = {
  readonly cutName: (key: string) => string;
  readonly targetName: (id: string) => string;
};

const list = new Intl.ListFormat("en", { type: "conjunction" });

// "Used for Director's Cut 2160p and 1080p", a cut before its Target as Needs you says it.
const fillWords =
  ({ cutName, targetName }: Names) =>
  (file: FilesListedFile) =>
    `Used for ${list.format(
      file.fills.map(({ targetId, edition }) =>
        edition === anyCut ? targetName(targetId) : `${cutName(edition)} ${targetName(targetId)}`,
      ),
    )}`;

function FileRow({
  file,
  about,
  subtitles,
  compact,
}: {
  file: FilesListedFile;
  about: string;
  subtitles: SubtitleFile | undefined;
  compact: boolean;
}) {
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
      detail={
        <>
          <FileMedia file={file} subtitles={subtitles && <SubtitleDetails file={subtitles} />} />
          {subtitles && <SubtitlesBlock file={subtitles} compact={compact} />}
        </>
      }
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
    ? ({ file, group: "kept", about: "Not used for any wanted Target" } as const)
    : ({ file, group: "filling", about: fills(file) } as const);
}

type Placed = ReturnType<typeof placed>;

/** What a file's subtitle rows share, for a file the subtitles read lists. */
type Subtitled = {
  readonly workId: string;
  readonly noun: string;
  readonly names: Names;
  readonly byFile: ReadonlyMap<string, SubtitlesFileSubtitles>;
  readonly providers: readonly SubtitlesProvider[];
};

function subtitleFileOf(file: FilesListedFile, at: Subtitled): SubtitleFile | undefined {
  const subtitles = at.byFile.get(file.id);
  const [fill] = file.fills;
  return subtitles === undefined
    ? undefined
    : {
        workId: at.workId,
        file,
        subtitles,
        providers: at.providers,
        noun: at.noun,
        version:
          fill === undefined ? "this file" : `the ${at.names.targetName(fill.targetId)} file`,
      };
}

function Rows({
  label,
  rows,
  at,
  compact,
}: {
  label: string;
  rows: readonly Placed[];
  at: Subtitled;
  compact: boolean;
}) {
  return (
    <List label={label}>
      {rows.map(({ file, about }) => (
        <FileRow
          key={file.id}
          file={file}
          about={about}
          subtitles={subtitleFileOf(file, at)}
          compact={compact}
        />
      ))}
    </List>
  );
}

const missingOf = (subtitles: SubtitlesFileSubtitles | undefined) =>
  subtitles?.wants.filter(
    (view) => rowStateOf(view) === "missing" || rowStateOf(view) === "waiting",
  ).length ?? 0;

const firstEpisode = (subtitles: SubtitlesFileSubtitles | undefined) =>
  subtitles?.episodes[0] ?? Number.POSITIVE_INFINITY;

/** A series' files in its seasons, the episodes in order, files with no season last. */
function seasonsOf(rows: readonly Placed[], at: Subtitled, label: string): Season[] {
  const bySeason = new Map<number | null, Placed[]>();
  for (const row of rows) {
    const season = at.byFile.get(row.file.id)?.season ?? null;
    bySeason.set(season, [...(bySeason.get(season) ?? []), row]);
  }
  return [...bySeason]
    .toSorted(([a], [b]) => (a ?? Number.POSITIVE_INFINITY) - (b ?? Number.POSITIVE_INFINITY))
    .map(([season, placedRows]) => {
      const ordered = placedRows.toSorted(
        (a, b) => firstEpisode(at.byFile.get(a.file.id)) - firstEpisode(at.byFile.get(b.file.id)),
      );
      return {
        season,
        files: ordered.length,
        missing: ordered.reduce((sum, { file }) => sum + missingOf(at.byFile.get(file.id)), 0),
        rows: () => (
          <Rows
            label={season === null ? label : `Season ${season} ${label.toLowerCase()}`}
            rows={ordered}
            at={at}
            compact
          />
        ),
      };
    });
}

function FileLists({ files, at }: { files: readonly FilesListedFile[]; at: Subtitled }) {
  const fills = fillWords(at.names);
  const rows = files.map((file) => placed(file, fills));
  const series = [...at.byFile.values()].some(({ season }) => season !== null);
  return (
    <div className="flex flex-col gap-3">
      {groups.map(({ group, label }) => {
        const listed = rows.filter((row) => row.group === group);
        if (listed.length === 0) {
          return null;
        }
        return series && group === "filling" ? (
          <SeasonFolds key={group} workId={at.workId} seasons={seasonsOf(listed, at, label)} />
        ) : (
          <Rows key={group} label={label} rows={listed} at={at} compact={series} />
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
  noun,
}: {
  listing: FilesFileList;
  root: WorksWorkRoot | null;
  names: Names;
  workId: string;
  /** The Work's noun for many, such as movies. */
  noun: string;
}) {
  const { byFile, providers } = useWorkSubtitles(workId);
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
        <p className="text-ink-3">
          No files yet. Downloaded files show here once they&apos;re imported.
        </p>
      ) : (
        <FileLists files={files} at={{ workId, noun, names, byFile, providers }} />
      )}
    </Section>
  );
}
