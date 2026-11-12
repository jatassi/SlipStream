import type { ImportsImportFile, ImportsImportFileOutcome } from "../api/gen/model";
import { Data } from "../ui/data";
import { nameOf } from "../works/paths";

const outcomeWords: { readonly [O in ImportsImportFileOutcome]: string } = {
  pending: "Reading",
  left_out: "Left out",
  not_better: "Not better than the file you have",
  waiting: "Waiting for you",
  sending: "Sending",
  placing: "Importing",
  placed: "Imported",
  declined: "Not imported",
  failed: "Failed",
};

/** A file's name, then where it got to and why. */
export function FileLine({ file }: { file: ImportsImportFile }) {
  return (
    <li className="flex min-w-0 flex-col gap-0.5">
      <span className="font-semibold text-ink-1 wrap-anywhere">
        <Data>{nameOf(file.path)}</Data>
      </span>
      <span className="text-ink-2">
        {outcomeWords[file.outcome]}
        {file.reason?.sentence === undefined || file.reason.sentence === null
          ? ""
          : `. ${file.reason.sentence}`}
      </span>
    </li>
  );
}
