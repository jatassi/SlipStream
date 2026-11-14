import type { ImportsImportEntry, WorksContentType } from "../api/gen/model";
import { useListNodesSuspense } from "../api/gen/query/nodes/nodes";
import { useGetWorkSuspense } from "../api/gen/query/works/works";
import { Data } from "../ui/data";
import { FileRow } from "./file-row";

/** The Node's name, as Pending imports names where an Import's files are. */
export function useNodeName(): (id: string | null) => string {
  const { nodes } = useListNodesSuspense().data;
  return (id) => nodes.find((node) => node.id === id)?.name ?? "A removed Node";
}

/** A Grab's Import's files that wait for the Admin. */
export const waitingFiles = (entry: ImportsImportEntry) =>
  entry.files.filter(({ outcome }) => outcome === "waiting");

function workIdOf(entry: ImportsImportEntry): string {
  if (entry.workId === null) {
    throw new Error(`the download Import ${entry.id} has no Work`);
  }
  return entry.workId;
}

/** "Severance.S01.1080p.WEB-DL", what the download is for and where it is, then its waiting files. */
export function DownloadEntry({
  entry,
  type,
}: {
  entry: ImportsImportEntry;
  type: WorksContentType;
}) {
  const work = useGetWorkSuspense(workIdOf(entry)).data;
  const nodeName = useNodeName();
  return (
    <li
      aria-label={entry.grab?.release ?? entry.folder}
      className="flex flex-col gap-3 rounded-card bg-glass p-4 ring-1 ring-hairline ring-inset"
    >
      <div className="flex min-w-0 flex-col gap-0.5">
        <span className="font-semibold text-ink-1 wrap-anywhere">
          <Data>{entry.grab?.release}</Data>
        </span>
        <span className="text-meta text-ink-3">
          {work.label} · Downloaded on {nodeName(entry.nodeId)}
        </span>
      </div>
      {entry.seeding && <p className="text-ink-2">Seeding continues while this waits.</p>}
      <ul aria-label="Files that wait" className="flex flex-col gap-4">
        {waitingFiles(entry).map((file) => (
          <FileRow key={file.position} entry={entry} file={file} type={type} items={work.items} />
        ))}
      </ul>
    </li>
  );
}
