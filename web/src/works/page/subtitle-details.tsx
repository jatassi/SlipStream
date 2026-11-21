import type {
  FilesListedFile,
  SubtitlesFileSubtitles,
  SubtitlesProvider,
  SubtitlesSidecarSource,
  SubtitlesTrack,
  SubtitlesWantView,
} from "../../api/gen/model";
import { useFileActions } from "../../subtitles/work-api";
import {
  type SubtitleFact,
  fitFacts,
  languageOf,
  providerName,
  wantLabel,
} from "../../subtitles/work-words";
import { Button } from "../../ui/button";
import { Data } from "../../ui/data";
import { Details } from "../../ui/details";
import { Fact, Facts } from "../../ui/facts";
import { Tooltip } from "../../ui/tooltip";
import { nameOf } from "../paths";

/** What every subtitle row of one file shares. */
export type SubtitleFile = {
  readonly workId: string;
  readonly file: FilesListedFile;
  readonly subtitles: SubtitlesFileSubtitles;
  readonly providers: readonly SubtitlesProvider[];
  /** The Work's noun for many, such as movies. */
  readonly noun: string;
  /** The file version in words, such as "the 2160p file". */
  readonly version: string;
};

export const stemOf = (path: string) => nameOf(path).replace(/\.[^.]+$/u, "");

/** A subtitle file's name, source and Timing sync numbers, under its row's Details. */
export function SidecarDetails({ view, file }: { view: SubtitlesWantView; file: SubtitleFile }) {
  const { sidecar } = view.state;
  if (sidecar === null) {
    return null;
  }
  const frameRate = file.file.media?.video.frameRate ?? 0;
  const facts: readonly SubtitleFact[] = [
    { label: "Name", value: `${stemOf(file.file.path)}.${sidecar.name}` },
    ...(sidecar.provider === null
      ? []
      : [{ label: "From", value: providerName(file.providers, sidecar.provider) }]),
    ...(sidecar.release === null ? [] : [{ label: "Release", value: sidecar.release }]),
    ...(sidecar.fit === null ? [] : fitFacts(sidecar.fit, frameRate)),
  ];
  return (
    <Details>
      <Facts look="well">
        {facts.map((fact) => (
          <Fact
            key={fact.label}
            label={
              fact.hover === undefined ? (
                fact.label
              ) : (
                <Tooltip label={fact.hover} name={`${fact.label}: ${fact.hover}`}>
                  <button
                    type="button"
                    className="cursor-help underline decoration-dotted underline-offset-2"
                  >
                    {fact.label}
                  </button>
                </Tooltip>
              )
            }
          >
            <Data>{fact.value}</Data>
          </Fact>
        ))}
      </Facts>
    </Details>
  );
}

const trackName = ({ language, forced, codec }: SubtitlesTrack) =>
  `${language === "" ? "Unknown language" : languageOf(language)}${forced ? " (forced)" : ""} · ${codec}`;

function trackWords(track: SubtitlesTrack, wants: readonly SubtitlesWantView[]): string {
  if (track.image) {
    return "An image subtitle, which can't count for a language.";
  }
  if (track.ignored) {
    return "You chose not to count this track.";
  }
  const met = wants.find(({ want }) => want === track.meets);
  return met === undefined
    ? "Not used for a wanted language."
    : `Counts for ${wantLabel(met.wanted)}.`;
}

function Track({ track, file }: { track: SubtitlesTrack; file: SubtitleFile }) {
  const actions = useFileActions(file.subtitles.fileId);
  const counting = track.meets !== null;
  return (
    <li className="flex flex-wrap items-center gap-x-3 gap-y-1">
      <span className="flex min-w-0 flex-1 flex-col">
        <Data className="text-ink-1">{trackName(track)}</Data>
        <span className="text-meta text-ink-3">{trackWords(track, file.subtitles.wants)}</span>
      </span>
      {track.ignored && (
        <Button size="small" onClick={() => void actions.count(track.stream)}>
          Count this track
        </Button>
      )}
      {counting && (
        <Button size="small" onClick={() => void actions.ignore(track.stream)}>
          Don&apos;t count this track
        </Button>
      )}
    </li>
  );
}

const sourceWords: { readonly [S in SubtitlesSidecarSource]: string } = {
  provider: "Downloaded",
  sibling: "Timed from another file's subtitle",
  found: "Found beside the file",
  upload: "Uploaded",
};

/** Every subtitle file beside the video, with what each is used for. */
function besideRows({ subtitles, providers, file }: SubtitleFile) {
  const stem = stemOf(file.path);
  const used = subtitles.wants.flatMap(({ wanted, state }) => {
    const { sidecar } = state;
    if (sidecar === null) {
      return [];
    }
    const from =
      sidecar.provider === null
        ? sourceWords[sidecar.source]
        : `${sourceWords[sidecar.source]} from ${providerName(providers, sidecar.provider)}`;
    return [{ name: `${stem}.${sidecar.name}`, words: `${from}. Used for ${wantLabel(wanted)}.` }];
  });
  const extra = subtitles.companions.map(({ name }) => ({
    name: `${stem}.${name}`,
    words: "Not used for a wanted language.",
  }));
  return [...used, ...extra];
}

/**
 * The Subtitles heading of a file's Details: its embedded tracks, each with
 * what it counts for and a way to stop counting it, then the subtitle files
 * beside the video.
 */
export function SubtitleDetails({ file }: { file: SubtitleFile }) {
  const { embedded } = file.subtitles;
  const beside = besideRows(file);
  if (embedded.length === 0 && beside.length === 0) {
    return null;
  }
  return (
    <div className="flex flex-col gap-2">
      <dt className="text-label text-ink-3">Subtitles</dt>
      <dd className="flex flex-col gap-3">
        {embedded.length > 0 && (
          <ul aria-label="Embedded subtitles" className="flex flex-col gap-2">
            {embedded.map((track) => (
              <Track key={track.stream} track={track} file={file} />
            ))}
          </ul>
        )}
        {beside.length > 0 && (
          <div className="flex flex-col gap-1">
            <span className="text-ink-2">Subtitle files beside this video</span>
            <ul aria-label="Subtitle files beside this video" className="flex flex-col gap-1">
              {beside.map(({ name, words }) => (
                <li key={name} className="flex flex-col">
                  <Data className="break-all text-ink-1">{name}</Data>
                  <span className="text-meta text-ink-3">{words}</span>
                </li>
              ))}
            </ul>
          </div>
        )}
      </dd>
    </div>
  );
}
