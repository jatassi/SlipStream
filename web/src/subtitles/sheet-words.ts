import type {
  SubtitlesCandidateReasonsItem,
  SubtitlesEnded,
  SubtitlesEndedKind,
  SubtitlesSidecar,
  SubtitlesTried,
  SubtitlesTriedOutcome,
  TargetsWantedLanguage,
} from "../api/gen/model";
import { agoWords } from "../format";
import { capital, subtitleOf } from "./work-words";

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

const endedLines: { readonly [K in SubtitlesEndedKind]: (ended: SubtitlesEnded) => string } = {
  added: () => "",
  failed_timing: ({ match, threshold }) =>
    match === null || threshold === null
      ? "Its timing didn't match this file."
      : `Its timing didn't match this file. Its timing match is ${match}%, and SlipStream keeps subtitles at ${threshold}% or more.`,
  image: () => "SlipStream can't use image subtitles.",
  unreadable: () => "SlipStream couldn't read this subtitle file.",
  wrong_episode: () => "This subtitle is for another episode.",
  marked_bad: () => "You marked this subtitle bad for this file.",
};

/** Why a download or an upload ended without a subtitle, for the sheet's alert. */
export const endedWords = (ended: SubtitlesEnded): string => endedLines[ended.kind](ended);

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

export const markedBadWords = (wanted: TargetsWantedLanguage) =>
  `${capital(subtitleOf(wanted))} marked bad. SlipStream is searching for another.`;

export const searchingWords = (wanted: TargetsWantedLanguage) =>
  `Searching for the ${subtitleOf(wanted)}.`;

/** The sheet's and dialog's title: what it looks for, and on which file. */
export const sheetTitle = (wanted: TargetsWantedLanguage, file: string) =>
  `${capital(subtitleOf(wanted))} for ${file}`;
