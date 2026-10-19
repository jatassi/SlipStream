import type { RootsMissingFiles, RootsProgress } from "../api/gen/model";
import { Data } from "../ui/data";
import { LiveBar } from "../ui/live-bar";
import { Spinner } from "../ui/spinner";
import { filesWords } from "./refused";
import { missingWords } from "./words";

const counted = new Intl.NumberFormat("en");

/** The re-home's check of every recorded file at the new place, read from the root's row. */
export function RehomeProgress({ progress }: { progress: RootsProgress | undefined }) {
  if (progress === undefined || progress.total === 0) {
    return (
      <p className="flex items-center gap-2 text-ink-2">
        <Spinner />
        Checking files at the new place…
      </p>
    );
  }
  const words = `Checked ${counted.format(progress.checked)} of ${filesWords(progress.total)} at the new place`;
  return (
    <div className="flex flex-col gap-2 text-ink-2">
      <p>{words}</p>
      <LiveBar label={words} value={(progress.checked / progress.total) * 100} />
    </div>
  );
}

export function MissingFiles({ missing, label }: { missing: RootsMissingFiles; label: string }) {
  const more = missing.count - missing.first.length;
  return (
    <div className="flex flex-col gap-2 rounded-card bg-warn-card px-4 py-3 text-warn-text ring-1 ring-warn-line ring-inset">
      <p className="font-semibold">
        {filesWords(missing.count)} {missing.count === 1 ? "isn't" : "aren't"} at the new place.
      </p>
      <ul aria-label="Missing files" className="flex flex-col gap-1">
        {missing.first.map((file) => (
          <li key={file.path} className="flex flex-wrap gap-x-2">
            <Data className="break-all">{file.path}</Data>
            <span className="text-warn-dim">{missingWords[file.reason]}</span>
          </li>
        ))}
      </ul>
      {more > 0 && <p className="text-warn-dim">And {counted.format(more)} more.</p>}
      <p className="text-warn-dim">
        Check the folder, or save anyway: SlipStream forgets those files, so their {label} read
        missing and it searches for them again.
      </p>
    </div>
  );
}
