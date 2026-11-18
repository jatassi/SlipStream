import type {
  SubtitlesCandidateReasonsItem,
  SubtitlesEnded,
  SubtitlesEndedKind,
  SubtitlesOutcome,
  SubtitlesOutcomeKind,
  SubtitlesProvider,
  SubtitlesSidecar,
  SubtitlesTried,
  SubtitlesTriedOutcome,
  TargetsWantedLanguage,
} from "../api/gen/model";
import { type Clock, agoWords, laterAtWords } from "../format";
import { capital, providerName, subtitleOf } from "./work-words";

export const reasonWords: { readonly [R in SubtitlesCandidateReasonsItem]: string } = {
  made_by_person: "Made by a person",
  same_group: "Same Release group",
  same_source: "Same source and resolution",
  provider_order: "Higher on your provider list",
};

const triedLines: {
  readonly [O in SubtitlesTriedOutcome]: (at: string, now: Date) => string;
} = {
  passed: () => "Its timing matched this file.",
  failed_timing: () => "Its timing didn't match this file.",
  marked_bad: (at, now) => `You marked this bad ${agoWords(at, now)}.`,
  image: () => "SlipStream can't use image subtitles.",
  unreadable: () => "SlipStream couldn't read this subtitle file.",
  wrong_episode: () => "It's for another episode.",
  too_large: () => "It's larger than 5 MB.",
};

/** A candidate's earlier try for this file, as Tried before lists it. */
export const triedWords = ({ outcome, at }: SubtitlesTried, now: Date): string =>
  triedLines[outcome](at, now);

export const matchWords = (match: number) => `Timing match ${match}%`;

/** What a pick or an upload is doing now. */
export const stepWords = {
  downloading: (provider: string) => `Downloading from ${provider}…`,
  checking: "Checking timing…",
};

export const addedWords = (wanted: TargetsWantedLanguage) =>
  `${capital(subtitleOf(wanted))} added.`;

type OutcomeLines = {
  readonly [K in SubtitlesOutcomeKind]: (
    name: string,
    until: string | null,
    clock: Clock,
  ) => string;
};

const outWords = (name: string, until: string | null, clock: Clock) =>
  until === null
    ? `${name} is out of downloads.`
    : `${name} is out of downloads. SlipStream searches it again ${laterAtWords(until, clock)}.`;

const searchLines: OutcomeLines = {
  ok: () => "",
  not_found: (name) => `${name} found nothing.`,
  refused: (name) => `${name} rejected the key. Replace it in Settings > Subtitles.`,
  limited: outWords,
  capped: outWords,
  unreachable: (name) => `${name} isn't responding.`,
};

/** How one provider's search ended, where it found nothing or couldn't search. */
export const outcomeWords = (
  { kind, provider, until }: SubtitlesOutcome,
  providers: readonly SubtitlesProvider[],
  clock: Clock,
) => searchLines[kind](providerName(providers, provider), until, clock);

const downloadLines: OutcomeLines = {
  ...searchLines,
  not_found: (name) => `${name} no longer has this subtitle.`,
};

type EndedReading = {
  readonly providers: readonly SubtitlesProvider[];
  readonly clock: Clock;
};

const endedLines: {
  readonly [K in SubtitlesEndedKind]: (ended: SubtitlesEnded, reading: EndedReading) => string;
} = {
  added: () => "",
  failed_timing: ({ match, threshold }) =>
    match === null || threshold === null
      ? "Its timing didn't match this file."
      : `Its timing didn't match this file. Its timing match is ${match}%, and SlipStream keeps subtitles at ${threshold}% or more.`,
  image: () => "SlipStream can't use image subtitles.",
  unreadable: () => "SlipStream couldn't read this subtitle file.",
  wrong_episode: () => "This subtitle is for another episode.",
  marked_bad: () => "You marked this subtitle bad for this file.",
  unfetched: ({ answer }, { providers, clock }) =>
    answer === null
      ? "SlipStream couldn't download this subtitle. Pick another subtitle."
      : `${downloadLines[answer.kind](providerName(providers, answer.provider), answer.until, clock)} Pick another subtitle.`,
};

/** Why a download or an upload ended without a subtitle, for the sheet's alert. */
export const endedWords = (ended: SubtitlesEnded, reading: EndedReading): string =>
  endedLines[ended.kind](ended, reading);

export const tooLargeWords = "Use a subtitle file of 5 MB or smaller.";

export const uploadFailedWords =
  "This subtitle's timing doesn't match the file, so SlipStream didn't keep it. Pick another file, or open Details to keep it anyway.";

export const keepAnywayWords = (match: number | null, threshold: number | null) =>
  match === null || threshold === null
    ? "If you keep it, its lines may show early or late."
    : `Its timing match is ${match}%, and SlipStream keeps subtitles at ${threshold}% or more. If you keep it, its lines may show early or late.`;

/** The Mark bad question, naming the file version, and what it does to an Admin's own file. */
export function markBadQuestion(
  wanted: TargetsWantedLanguage,
  sidecar: SubtitlesSidecar,
  { version, stem }: { version: string; stem: string },
): { title: string; body: string } {
  if (sidecar.source === "upload" || sidecar.source === "found") {
    return {
      title: `Mark ${stem}.${sidecar.name} bad?`,
      body: `SlipStream didn't download it, so this deletes your file and can't be undone. SlipStream then searches for another ${subtitleOf(wanted)}.`,
    };
  }
  return {
    title: `Mark the ${subtitleOf(wanted)} on ${version} bad?`,
    body: "SlipStream deletes it, won't use it for this file again and searches for another.",
  };
}

const turnOnWords = (what: string) =>
  `Subtitle search is off. Turn it on in Settings > Subtitles to search for ${what}.`;

/** Mark bad's toast, which says a search runs only while Subtitle search is on. */
export const markedBadWords = (wanted: TargetsWantedLanguage, searchOn: boolean) =>
  `${capital(subtitleOf(wanted))} marked bad. ${searchOn ? "SlipStream is searching for another." : turnOnWords("another")}`;

/** Search's toast, which says a search runs only while Subtitle search is on. */
export const searchingWords = (wanted: TargetsWantedLanguage, searchOn: boolean) =>
  searchOn ? `Searching for the ${subtitleOf(wanted)}.` : turnOnWords(`the ${subtitleOf(wanted)}`);

/** The sheet's and dialog's title: what it looks for, and on which file. */
export const sheetTitle = (wanted: TargetsWantedLanguage, file: string) =>
  `${capital(subtitleOf(wanted))} for ${file}`;
