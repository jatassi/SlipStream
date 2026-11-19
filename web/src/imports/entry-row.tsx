import type { WorksContentType } from "../api/gen/model";
import type { RootEntry } from "./root-entry";
import { bytes } from "../format";
import { Quality } from "../release/quality";
import { Chip } from "../ui/chip";
import { Data } from "../ui/data";
import { Poster } from "../ui/poster";
import { EntryActions } from "./entry-actions";
import { EntryDetails } from "./entry-details";
import { filesWords, named, reasonsOf } from "./words";

function Suggestion({ entry }: { entry: RootEntry }) {
  const { suggestion } = entry;
  if (suggestion === null) {
    return <p className="text-ink-2">No match yet.</p>;
  }
  return (
    <p className="flex flex-wrap items-center gap-2">
      <span className="text-ink-2">Suggested match</span>
      <span className="font-semibold">{named(suggestion.label, suggestion.year)}</span>
      {suggestion.workId !== null && <Chip>In the library</Chip>}
    </p>
  );
}

/** One folder that waits: why, its suggested match, what to do, and the files under Details. */
export function EntryRow({ entry, type }: { entry: RootEntry; type: WorksContentType }) {
  const size = entry.files.reduce((total, file) => total + file.size, 0);
  const qualities = [
    ...new Set(entry.files.flatMap((file) => (file.quality === null ? [] : [file.quality]))),
  ];
  return (
    <li
      aria-label={entry.folder}
      className="flex gap-3 rounded-card bg-glass p-4 ring-1 ring-hairline ring-inset"
    >
      <Poster url={entry.suggestion?.posterUrl ?? undefined} />
      <div className="flex min-w-0 flex-1 flex-col items-start gap-2">
        <div className="flex min-w-0 flex-col gap-0.5">
          <span className="font-semibold text-ink-1 wrap-anywhere">
            <Data>{entry.folder}</Data>
          </span>
          <span className="text-meta text-ink-3">
            {filesWords(entry.files.length)} · {bytes(size)}
            {qualities.map((quality) => (
              <span key={quality}>
                {" · "}
                <Quality quality={quality} look="word" />
              </span>
            ))}
          </span>
        </div>
        {reasonsOf(entry, type).map((sentence) => (
          <p key={sentence}>{sentence}</p>
        ))}
        <Suggestion entry={entry} />
        <EntryActions entry={entry} type={type} />
        <EntryDetails entry={entry} type={type} />
      </div>
    </li>
  );
}
