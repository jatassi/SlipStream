import type { ImportsImportEntry, WorksContentType } from "../api/gen/model";
import { useGetWorkSuspense } from "../api/gen/query/works/works";
import { Data } from "../ui/data";
import { Empty } from "../ui/empty";
import { DownloadEntry, waitingFiles } from "./download-entry";
import { FileRow } from "./file-row";

/** "From a Conversion of Heat.1995.2160p.UHD.BluRay.mkv", where it was made from. */
export const madeWords = (entry: ImportsImportEntry) =>
  entry.conversion === null
    ? "From a Conversion"
    : `From a Conversion of ${entry.conversion.sourceFile}`;

function workIdOf(entry: ImportsImportEntry): string {
  if (entry.workId === null) {
    throw new Error(`the conversion Import ${entry.id} has no Work`);
  }
  return entry.workId;
}

/** A conversion Import: where its file was made from and its Work, then its waiting file. */
export function ConversionEntry({
  entry,
  type,
}: {
  entry: ImportsImportEntry;
  type: WorksContentType;
}) {
  const work = useGetWorkSuspense(workIdOf(entry)).data;
  return (
    <li
      aria-label={madeWords(entry)}
      className="flex flex-col gap-3 rounded-card bg-glass p-4 ring-1 ring-hairline ring-inset"
    >
      <div className="flex min-w-0 flex-col gap-0.5">
        <span className="font-semibold text-ink-1 wrap-anywhere">
          {entry.conversion === null ? (
            madeWords(entry)
          ) : (
            <>
              From a Conversion of <Data>{entry.conversion.sourceFile}</Data>
            </>
          )}
        </span>
        <span className="text-meta text-ink-3">{work.label}</span>
      </div>
      <ul aria-label="Files that wait" className="flex flex-col gap-4">
        {waitingFiles(entry).map((file) => (
          <FileRow key={file.position} entry={entry} file={file} type={type} items={work.items} />
        ))}
      </ul>
    </li>
  );
}

/** A Pending import's own page title: its Release, its folder, or where its file was made from. */
export const entryTitle = (entry: ImportsImportEntry) =>
  entry.source === "conversion" ? madeWords(entry) : (entry.grab?.release ?? entry.folder);

/** One download's or Conversion's Pending import on its own page: its waiting files, or that none wait. */
export function PendingEntry({
  entry,
  type,
}: {
  entry: ImportsImportEntry;
  type: WorksContentType | undefined;
}) {
  if (type === undefined || waitingFiles(entry).length === 0) {
    return (
      <Empty look="sheet">
        {entry.source === "conversion"
          ? "Nothing in this file needs attention now."
          : "Nothing in this download needs attention now."}
      </Empty>
    );
  }
  return (
    <ul>
      {entry.source === "conversion" ? (
        <ConversionEntry entry={entry} type={type} />
      ) : (
        <DownloadEntry entry={entry} type={type} />
      )}
    </ul>
  );
}
