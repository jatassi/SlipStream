import type {
  SubtitlesFit,
  SubtitlesLastSearch,
  SubtitlesProvider,
  SubtitlesProviderKindsItem,
  SubtitlesSidecar,
  SubtitlesSidecarSource,
  SubtitlesWantStateMissing,
  SubtitlesWantView,
  TargetsWantedLanguage,
} from "../api/gen/model";
import { type Clock, agoWords, atWords, countWords, laterAtWords } from "../format";
import { waitsForScan } from "./scan-words";

const languageNames = new Intl.DisplayNames(["en"], { type: "language" });

/** "English", or "Brazilian Portuguese" for pt-BR. */
export const languageOf = (code: string): string => languageNames.of(code) ?? code;

/** A Wanted Language's name, as its chip and row label read: "English" or "English, forced". */
export const wantLabel = ({ language, forced }: TargetsWantedLanguage): string =>
  `${languageOf(language)}${forced ? ", forced" : ""}`;

/** "English subtitle" or "forced English subtitle", inside a sentence. */
export const subtitleOf = ({ language, forced }: TargetsWantedLanguage): string =>
  `${forced ? "forced " : ""}${languageOf(language)} subtitle`;

export const capital = (words: string) => `${words.charAt(0).toUpperCase()}${words.slice(1)}`;

const subtitles = { one: "subtitle", many: "subtitles" };

/** What a row's sentence reads beyond the want itself. */
export type Reading = {
  readonly clock: Clock;
  readonly providers: readonly SubtitlesProvider[];
  /** The Work's noun for many, such as movies. */
  readonly noun: string;
  /** What the providers search for this file: a film, or an episode. */
  readonly kind: SubtitlesProviderKindsItem;
  /** The file has no audio track, so its audio can't be scanned. */
  readonly noAudio: boolean;
};

export const providerName = (providers: readonly SubtitlesProvider[], id: string): string =>
  providers.find((provider) => provider.id === id)?.name ?? id;

const againAt = (next: string | null, clock: Clock) =>
  next === null ? "" : ` SlipStream searches again ${laterAtWords(next, clock)}.`;

const either = new Intl.ListFormat("en", { type: "disjunction" });
const both = new Intl.ListFormat("en", { type: "conjunction" });

export const noHolderWords =
  "No Node has the Subtitles Role, so SlipStream can't download or time subtitles. Assign it on the Nodes page.";

// The Health item's sentence for a Content Type no provider searches, with the same fix.
function uncoveredWords({ providers, noun, kind }: Reading): string {
  const covering = providers.filter((provider) => provider.kinds.includes(kind));
  const keyless = covering.filter((provider) => provider.needsKey && !provider.hasKey);
  const off = covering.filter((provider) => !(provider.needsKey && !provider.hasKey));
  const fix =
    off.length > 0
      ? `Turn on ${either.format(off.map(({ name }) => name))} in Settings > Subtitles.`
      : `Add a ${either.format(keyless.map(({ name }) => name))} key.`;
  return `No Subtitle Provider that is on has subtitles for ${noun.toLowerCase()}. ${fix}`;
}

function pausedWords({ providers, kind, clock }: Reading): string {
  const out = providers.flatMap((provider) =>
    provider.kinds.includes(kind) && provider.state.until !== null
      ? [`${provider.name}'s limit resets ${laterAtWords(provider.state.until, clock)}`]
      : [],
  );
  return out.length === 0
    ? "Waiting for the Subtitle Providers' limits to reset."
    : `Waiting for the Subtitle Providers' limits to reset. ${both.format(out)}.`;
}

const foundWords = (last: SubtitlesLastSearch | null) =>
  last === null ? "Subtitles were found" : `${countWords(last.found, subtitles)} were found`;

function dueWords(next: string | null, clock: Clock): string {
  if (next === null || new Date(next) <= clock.now) {
    return "Due in the next Subtitle search.";
  }
  const at = atWords(next, clock);
  return `Due in the next Subtitle search, ${at.charAt(0).toLowerCase()}${at.slice(1)}.`;
}

const noAudioWords =
  "This file has no audio track, so Timing sync can't run. Upload a subtitle and keep it anyway under Details.";

const missingLines: {
  readonly [M in NonNullable<SubtitlesWantStateMissing>]: (
    view: SubtitlesWantView,
    reading: Reading,
  ) => string;
} = {
  no_holder: () => noHolderWords,
  no_provider: (_view, reading) => uncoveredWords(reading),
  paused: (_view, reading) => pausedWords(reading),
  no_aired_number: () =>
    "TVDB lists no aired number for this episode, so the Subtitle Providers can't be searched for it. Upload a subtitle instead.",
  map_failed: ({ state }, { noAudio, clock }) =>
    noAudio
      ? noAudioWords
      : `SlipStream couldn't read this file's audio.${state.next === null ? "" : ` It tries again ${laterAtWords(state.next, clock)}.`}`,
  map_due: (_view, { noAudio }) => (noAudio ? noAudioWords : waitsForScan),
  none_passed: ({ state, last }, { clock }) =>
    `${foundWords(last)}, and none matched this file's timing.${state.next === null ? " You can upload one." : ` SlipStream searches again ${laterAtWords(state.next, clock)}, or you can upload one.`}`,
  nothing_found: ({ state, last }, { clock }) =>
    `No subtitle was found${last === null ? "" : ` ${agoWords(last.at, clock.now)}`}.${againAt(state.next, clock)}`,
  never_searched: ({ state }, { clock }) => dueWords(state.next, clock),
};

const hindiWords = (sidecar: SubtitlesSidecar) =>
  ` Jellyfin reads .hi as Hindi. Rename the file to .${sidecar.name.replace(".hi.", ".sdh.")} to fix that.`;

const metLines: {
  readonly [S in SubtitlesSidecarSource]: (sidecar: SubtitlesSidecar, from: string) => string;
} = {
  provider: ({ final }, from) =>
    final
      ? `From ${from}. A person made it for this file's Release group, so Subtitle search doesn't look for a better one.`
      : `From ${from}. Subtitle search looks for a better match.`,
  sibling: () => "Timed from another file's subtitle.",
  found: ({ fit }) =>
    fit === null
      ? "Found beside the file. SlipStream hasn't checked its timing."
      : "Found beside the file. Its timing matches this file.",
  upload: ({ fit }) =>
    fit === null
      ? "Uploaded and kept without a timing check."
      : "Uploaded. Its timing matches this file.",
};

const metWords = (sidecar: SubtitlesSidecar, providers: readonly SubtitlesProvider[]) =>
  sidecar.timingDue
    ? "Checking this subtitle's timing against the new file."
    : metLines[sidecar.source](sidecar, providerName(providers, sidecar.provider ?? ""));

/** The one sentence a want's row leads with. */
export function stateLine(view: SubtitlesWantView, reading: Reading): string {
  const { state, wanted } = view;
  const language = languageOf(wanted.language);
  if (state.kind === "missing" && state.missing !== null) {
    return missingLines[state.missing](view, reading);
  }
  if (state.kind === "not_needed") {
    return state.why === "audio_in_language"
      ? `Not needed. The file has ${language} audio.`
      : `Not needed. The full ${language} subtitle includes the forced lines.`;
  }
  if (state.sidecar !== null) {
    const words = metWords(state.sidecar, reading.providers);
    return state.sidecar.hindiWarning ? `${words}${hindiWords(state.sidecar)}` : words;
  }
  return "Embedded in the file.";
}

/** A row's state, as its chip and mark show it. */
export type RowState = "met" | "embedded" | "not_needed" | "waiting" | "missing";

const waits: ReadonlySet<SubtitlesWantStateMissing> = new Set([
  "map_due",
  "paused",
  "never_searched",
]);

export function rowStateOf({ state }: SubtitlesWantView): RowState {
  if (state.kind === "missing") {
    return waits.has(state.missing) ? "waiting" : "missing";
  }
  return state.kind;
}

export const rowWords: { readonly [S in RowState]: string } = {
  met: "Met",
  embedded: "Embedded",
  not_needed: "Not needed",
  waiting: "Waiting",
  missing: "Missing",
};

/** One timing fact under a subtitle's Details. */
export type Fact = { readonly label: string; readonly value: string; readonly hover?: string };

const seconds = new Intl.NumberFormat("en", {
  minimumFractionDigits: 1,
  maximumFractionDigits: 1,
  signDisplay: "always",
});
const rate = new Intl.NumberFormat("en", { maximumFractionDigits: 3 });

function framerateWords(ratio: number, frameRate: number): string {
  if (ratio === 1) {
    return "Same as the file";
  }
  return frameRate === 0
    ? `Scaled by ${rate.format(ratio)}`
    : `${rate.format(frameRate * ratio)} to ${rate.format(frameRate)} fps`;
}

/** The Timing sync numbers, with the Splits hover, as the spec's Details list reads them. */
export function fitFacts(fit: SubtitlesFit, frameRate: number): readonly Fact[] {
  return [
    { label: "Offset", value: `${seconds.format(fit.offsetMs / 1000)} s` },
    { label: "Framerate", value: framerateWords(fit.ratio, frameRate) },
    {
      label: "Splits found",
      value: String(fit.segments - 1),
      hover: "Places where the subtitle's timing jumps, such as an ad break.",
    },
    { label: "Timing match", value: `${fit.match}%` },
  ];
}
