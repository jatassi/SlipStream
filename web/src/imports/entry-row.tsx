import type { ImportsEntry, ImportsHint, WorksContentType } from "../api/gen/model";
import { bytes } from "../format";
import { useQualityNames } from "../targets/quality-names";
import { Chip } from "../ui/chip";
import { Data } from "../ui/data";
import { Details } from "../ui/details";
import { Poster } from "../ui/poster";
import { EntryActions } from "./entry-actions";
import { filesWords, named, reasonOf, reasonsOf } from "./words";

const providers = { tmdb: "TMDB", imdb: "IMDb", tvdb: "TVDB" } as const;

const hintWords = ({ title, year, ids }: ImportsHint) =>
  [named(title, year), ...ids.map(({ provider, value }) => `${providers[provider]} ${value}`)].join(
    " · ",
  );

function Suggestion({ entry }: { entry: ImportsEntry }) {
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

function EntryDetails({ entry, type }: { entry: ImportsEntry; type: WorksContentType }) {
  const quality = useQualityNames();
  return (
    <Details>
      <dl className="mt-1 grid grid-cols-[max-content_minmax(0,1fr)] gap-x-4 gap-y-1 rounded-item bg-well px-3 py-2">
        <dt>Name reads</dt>
        <dd>{hintWords(entry.hint)}</dd>
        {entry.files.map((file) => (
          <div key={file.path} className="contents">
            <dt>
              <Data>{file.path}</Data>
            </dt>
            <dd className="wrap-anywhere">
              <Data>
                {file.quality === null ? "No quality read" : quality(file.quality)} ·{" "}
                {bytes(file.size)}
              </Data>{" "}
              {reasonOf(file.reason, type)}
              {file.reason.line !== undefined && (
                <span className="block">
                  <Data>{file.reason.line}</Data>
                </span>
              )}
            </dd>
          </div>
        ))}
      </dl>
    </Details>
  );
}

/** One folder that waits: why, its suggested match, what to do, and the files under Details. */
export function EntryRow({ entry, type }: { entry: ImportsEntry; type: WorksContentType }) {
  const quality = useQualityNames();
  const size = entry.files.reduce((total, file) => total + file.size, 0);
  const qualities = [
    ...new Set(
      entry.files.flatMap((file) => (file.quality === null ? [] : [quality(file.quality)])),
    ),
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
            {[filesWords(entry.files.length), bytes(size), ...qualities].join(" · ")}
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
