import type {
  GrabsReason,
  GrabsReasonAttribute,
  GrabsReasonCode,
  TargetsTarget,
} from "../api/gen/model";

/** The names a reason's words read: each Target by id, and each quality's name by key. */
export type ReasonNames = {
  readonly targets: ReadonlyMap<string, Pick<TargetsTarget, "name" | "scores">>;
  readonly qualities: ReadonlyMap<string, string>;
};

const attributeWords: Record<GrabsReasonAttribute, string> = {
  hdr: "HDR",
  video_codec: "video codec",
  audio_codec: "audio codec",
  channels: "channels",
  language: "language",
};

type Said = {
  readonly value: string;
  readonly reason: GrabsReason;
  /** The Target's name, or "a deleted Target" for one deleted since the run. */
  readonly at: string;
  readonly target: Pick<TargetsTarget, "name" | "scores"> | undefined;
  readonly qualities: ReadonlyMap<string, string>;
};

const capital = (words: string): string => `${words.charAt(0).toUpperCase()}${words.slice(1)}`;

// The same words as the tester's, which grabs.reasonWords writes in Go.
const words: Record<GrabsReasonCode, (said: Said) => string> = {
  wrong_work: () => "An id in the name belongs to another Work.",
  wrong_item: ({ value }) =>
    value === "" ? "It names none of the Work's Items." : `The Work has no Item ${value}.`,
  covers_more: () => "It holds more than one Item.",
  wrong_title: ({ value }) =>
    value === "" ? "Title doesn't match." : `Title doesn't match: ${value}.`,
  cut_not_wanted: ({ value, at }) => `The Work doesn't want the ${value} cut at ${at}.`,
  quality_not_in_target: ({ value, at, qualities }) =>
    `${capital(at)} doesn't accept ${value === "" ? "a release with no Quality" : (qualities.get(value) ?? value)}.`,
  filter: ({ reason, at }) =>
    `It fails the ${reason.attribute === undefined ? "" : `${attributeWords[reason.attribute]} `}filter of ${at}.`,
  size_below: ({ at }) => `It's smaller than ${at} allows.`,
  size_above: ({ at }) => `It's larger than ${at} allows.`,
  size_not_checked: ({ at }) =>
    `Its size isn't checked at ${at} because the name gives no size or the runtime is unknown.`,
  score_below_minimum: ({ value, at, target }) =>
    target === undefined
      ? `Score ${value} is below the minimum of ${at}.`
      : `Score ${value} is below the minimum of ${target.scores.minimum} at ${at}.`,
  not_an_upgrade: ({ at }) => `It isn't an upgrade of the file at ${at}.`,
  repack_other_group: () =>
    "This repack is from another release group than the file it would replace.",
};

/** A reason as a sentence. */
export function reasonWords(reason: GrabsReason, { targets, qualities }: ReasonNames): string {
  const target = reason.targetId === undefined ? undefined : targets.get(reason.targetId);
  return words[reason.code]({
    value: reason.value ?? "",
    reason,
    at: target?.name ?? "a deleted Target",
    target,
    qualities,
  });
}

/** A note, such as size_not_checked, says something about a Release without refusing it. */
const isNote = (reason: GrabsReason): boolean => reason.code === "size_not_checked";

/**
 * What a refused Release's reasons add up to: a hard refusal keeps every
 * search from grabbing it, and soft ones keep only an automatic search.
 */
export function refusalWords(reasons: readonly GrabsReason[]): string {
  if (reasons.some((reason) => !reason.soft && !isNote(reason))) {
    return "No search can grab it.";
  }
  if (reasons.some((reason) => reason.soft)) {
    return "An automatic search skips it. A Manual search can grab it.";
  }
  return "No Target this search was for can take it.";
}
