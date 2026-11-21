import type {
  SubtitlesLead,
  SubtitlesProvider,
  TargetsWantedLanguage,
  WorksContentType,
} from "../api/gen/model";
import { everyWords } from "../grabs/settings-section";
import type { Clock } from "../ui/clock";
import { whenWords } from "../ui/time-words";

const counted = new Intl.NumberFormat("en");
const list = new Intl.ListFormat("en", { type: "conjunction" });
const languageNames = new Intl.DisplayNames(["en"], { type: "language" });

/** A subtitle language's English name: "English", "Brazilian Portuguese". */
export const languageName = (code: string): string => languageNames.of(code) ?? code;

// The languages the Profile editor offers, by the codes people pick most.
// Any ISO 639 code is valid on the wire.
const offered =
  "en fr de es it pt pt-BR nl sv no da fi pl cs hu ro el tr ru uk he ar fa hi ta te th vi id ja ko zh".split(
    " ",
  );

export const subtitleLanguages: readonly { readonly code: string; readonly name: string }[] =
  offered.map((code) => ({ code, name: languageName(code) }));

type Want = TargetsWantedLanguage;
type Hearing = Want["hearingImpaired"];

export const kindWords = (forced: boolean): string => (forced ? "forced only" : "full");

export const hearingWords: Readonly<Record<Hearing, string>> = {
  avoid: "avoided",
  allow: "allowed",
  prefer: "preferred",
};

export const audioWords = (want: Pick<Want, "audio" | "language">): string =>
  want.audio === "always" ? "Always" : `Without ${languageName(want.language)} audio`;

/** The Profile editor's sentence: "Always get English subtitles, full, hearing impaired allowed." */
export function wantSentence(want: Want): string {
  const name = languageName(want.language);
  const when = want.audio === "always" ? "Always get" : `Without ${name} audio, get`;
  return `${when} ${name} subtitles, ${kindWords(want.forced)}, hearing impaired ${hearingWords[want.hearingImpaired]}.`;
}

const kindMeanings = {
  full: "Full: every line of dialogue.",
  forced: "Forced only: just the lines in another language, such as signs.",
};

const hearingMeanings: Readonly<Partial<Record<Hearing, string>>> = {
  avoid: "Avoided: used only when nothing else fits.",
  prefer: "Preferred: chosen first when one fits.",
};

/** What a Wanted Language's kind and hearing-impaired choice mean, under its sentence. */
export const wantMeanings = (want: Want): string =>
  [kindMeanings[want.forced ? "forced" : "full"], hearingMeanings[want.hearingImpaired]]
    .filter((line) => line !== undefined)
    .join(" ");

export const duplicateWords = (want: Pick<Want, "forced" | "language">): string =>
  `${want.forced ? "Forced-only" : "Full"} ${languageName(want.language)} subtitles are already on this list.`;

/** "English, full, hearing impaired allowed", with ", without English audio" for a want only then. */
export function wantPhrase(want: Want): string {
  const name = languageName(want.language);
  const phrase = `${name}, ${kindWords(want.forced)}, hearing impaired ${hearingWords[want.hearingImpaired]}`;
  return want.audio === "always" ? phrase : `${phrase}, without ${name} audio`;
}

type Kind = SubtitlesProvider["kinds"][number];

/** What a Content Type's files are searched as: episodes for a type with levels, else films. */
export const kindOfType = (type: Pick<WorksContentType, "levels">): Kind =>
  type.levels.length > 0 ? "episode" : "feature";

const typeNames = (types: readonly WorksContentType[]) => list.format(types.map((t) => t.name));

const provided = (provider: SubtitlesProvider, types: readonly WorksContentType[]) =>
  types.filter((type) => provider.kinds.includes(kindOfType(type)));

/** "TV only." for a provider that searches fewer of the enabled Content Types than exist. */
function scopeWords(provider: SubtitlesProvider, types: readonly WorksContentType[]) {
  const covered = provided(provider, types);
  return covered.length > 0 && covered.length < types.length
    ? `${typeNames(covered)} only.`
    : undefined;
}

const ordinals = ["first", "second", "third", "fourth", "fifth", "sixth"];

const placeWords = (position: number) => ordinals[position - 1] ?? `number ${String(position)}`;

function allowanceWords(provider: SubtitlesProvider): string | undefined {
  const downloads = provider.allowance?.downloads;
  if (downloads === undefined || downloads === null) {
    return undefined;
  }
  return `${counted.format(downloads.remaining)} of ${counted.format(downloads.limit)} downloads left`;
}

function resetWords(provider: SubtitlesProvider, clock: Clock): string {
  const downloads = provider.allowance?.downloads;
  return downloads === undefined || downloads === null
    ? ""
    : ` The count resets ${whenWords(downloads.resets, "next", clock)}.`;
}

const stateLines: Readonly<
  Record<
    SubtitlesProvider["state"]["kind"],
    (provider: SubtitlesProvider, types: readonly WorksContentType[], clock: Clock) => string
  >
> = {
  ready: (provider, _types, clock) => {
    const left = allowanceWords(provider);
    return left === undefined ? "Ready." : `Ready, with ${left}.${resetWords(provider, clock)}`;
  },
  needs_key: (provider, types) => {
    const covered = provided(provider, types);
    const search = covered.length === 0 ? "" : ` Add it to search ${typeNames(covered)}.`;
    return `${provider.name} needs your free key.${search}`;
  },
  out: ({ name, state }, _types, clock) =>
    state.until === null
      ? `${name} is out of downloads.`
      : `${name} is out of downloads. SlipStream searches it again ${whenWords(state.until, "next", clock)}.`,
  rejected: ({ name, state }, _types, clock) => {
    const at = state.since === null ? "" : ` ${whenWords(state.since, "at", clock)}`;
    return state.words === null
      ? `${name} rejected the key${at}. Replace the key.`
      : `${name} rejected the key${at} and responded "${state.words}". Replace the key.`;
  },
  failing: ({ name, state }, _types, clock) =>
    state.since === null
      ? `${name} isn't responding. SlipStream retries automatically.`
      : `${name} hasn't responded ${whenWords(state.since, "since", clock)}. SlipStream retries automatically.`,
  off: ({ name }) => `Off. SlipStream doesn't search ${name}.`,
};

/** A provider's row: its state, then which Content Types it alone covers and whether it needs a key. */
export function providerLine(
  provider: SubtitlesProvider,
  types: readonly WorksContentType[],
  clock: Clock,
): string {
  const keyless = provider.needsKey ? undefined : "No key needed.";
  const state =
    provider.state.kind === "ready" && keyless !== undefined
      ? undefined
      : stateLines[provider.state.kind](provider, types, clock);
  return [state, scopeWords(provider, types), keyless]
    .filter((part) => part !== undefined)
    .join(" ");
}

export const providersCaption =
  "SlipStream searches these providers in order. When two subtitles match the file equally well, SlipStream picks the one from the provider higher on the list.";

/** The key Dialog's answer once the provider accepts the key. */
export function acceptedWords(provider: SubtitlesProvider): string {
  const owner =
    provider.account === null
      ? undefined
      : `It belongs to ${provider.account}${provider.plan === null ? "" : ` on ${provider.name}'s ${provider.plan} plan`}`;
  const left = allowanceWords(provider);
  const belongs =
    owner === undefined ? undefined : left === undefined ? `${owner}.` : `${owner}, with ${left}.`;
  return [
    "The key works.",
    belongs,
    `${provider.name} is ${placeWords(provider.position)} in the search order.`,
  ]
    .filter((part) => part !== undefined)
    .join(" ");
}

/** The three steps of the key Dialog, each a sentence, with the key page on the first. */
export function keySteps(provider: SubtitlesProvider): readonly [string, string, string] {
  return provider.id === "subsource"
    ? [
        `Open My Profile on ${provider.name}. Sign in, or create a free account.`,
        "Open the API tab, generate a key and copy it.",
        "Paste the key here.",
      ]
    : [
        `Open ${provider.name}'s API page. Sign in, or create a free account.`,
        "Generate a key and copy it.",
        "Paste the key here.",
      ];
}

export const passwordWords = (provider: SubtitlesProvider): string =>
  `SlipStream never asks for your ${provider.name} password.`;

/** The provider that the lead's button offers a key for: the first in order that needs one and covers an uncovered type. */
function keyToAdd(
  providers: readonly SubtitlesProvider[],
  uncovered: readonly WorksContentType[],
): SubtitlesProvider | undefined {
  return providers.find(
    (provider) =>
      provider.needsKey &&
      !provider.hasKey &&
      uncovered.some((type) => provider.kinds.includes(kindOfType(type))),
  );
}

type Lead = {
  readonly sentences: readonly string[];
  /** The Admin's next step: open Profiles, or add this provider's key. */
  readonly next: "profiles" | SubtitlesProvider | undefined;
};

type LeadOf = (
  lead: SubtitlesLead,
  profiles: number,
  providers: readonly SubtitlesProvider[],
  types: readonly WorksContentType[],
) => Lead;

function uncoveredLead(
  lead: SubtitlesLead,
  providers: readonly SubtitlesProvider[],
  types: readonly WorksContentType[],
): Lead {
  const uncovered = types.filter((type) => lead.uncovered.includes(type.id));
  const add = keyToAdd(providers, uncovered);
  const searching = providers
    .filter((provider) => !provider.needsKey && provider.state.kind === "ready")
    .flatMap((provider) => {
      const covered = provided(provider, types).filter((type) => !uncovered.includes(type));
      return covered.length === 0
        ? []
        : [`SlipStream already searches ${provider.name} for ${typeNames(covered)}.`];
    });
  return {
    sentences: [
      `${typeNames(uncovered)} ${uncovered.length === 1 ? "has" : "have"} no Subtitle Provider.`,
      ...(add === undefined ? [] : [`Add a free ${add.name} key, which takes about a minute.`]),
      ...searching,
    ],
    next: add,
  };
}

function readyLead(lead: SubtitlesLead, profiles: number): Lead {
  const on = `Subtitles are on for ${String(profiles)} ${profiles === 1 ? "Profile" : "Profiles"}.`;
  const missing =
    lead.missing === 0
      ? "No subtitles are missing"
      : `${counted.format(lead.missing)} ${lead.missing === 1 ? "subtitle is" : "subtitles are"} missing`;
  return {
    sentences: [on, `${missing}, and Subtitle search runs ${everyWords(lead.interval)}.`],
    next: undefined,
  };
}

const leads: Readonly<Record<SubtitlesLead["kind"], LeadOf>> = {
  no_languages: () => ({
    sentences: ["No Profile has a subtitle language yet. Add a language to a Profile to start."],
    next: "profiles",
  }),
  uncovered: (lead, _profiles, providers, types) => uncoveredLead(lead, providers, types),
  off: () => ({
    sentences: ["Subtitle search is off. Turn it on to fetch missing subtitles."],
    next: undefined,
  }),
  ready: (lead, profiles) => readyLead(lead, profiles),
};

/** The Subtitles section's lead: what holds first, then the Admin's next step. */
export const leadOf: LeadOf = (lead, ...rest) => leads[lead.kind](lead, ...rest);
