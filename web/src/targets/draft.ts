import type {
  TargetsFilterMode,
  TargetsNewTarget,
  TargetsQuality,
  TargetsQualityList,
  TargetsTarget,
} from "../api/gen/model";
import {
  type Layout,
  type Order,
  type QualityKey,
  cam,
  keepCutoff,
  layoutOf,
  orderOf,
  presetOrder,
} from "./order";
import { type Attribute, attributes } from "./vocab";

export type Rule =
  | { readonly mode: "any" }
  | { readonly mode: TargetsFilterMode; readonly values: readonly string[] };

/** A size as typed, in MiB per minute; an empty maximum is no limit. */
export type SizeDraft = { readonly min: string; readonly max: string };

/**
 * A Target being edited. The order holds every quality but CAM, accepted
 * or not, so accepting one puts it where it last stood; sizes are kept for
 * every quality and only the accepted ones are sent.
 */
export type TargetDraft = {
  readonly name: string;
  readonly order: Order;
  readonly accepted: ReadonlySet<QualityKey>;
  readonly filters: Readonly<Record<Attribute, Rule>>;
  readonly sizes: Readonly<Partial<Record<QualityKey, SizeDraft>>>;
};

const anyRules = (): Record<Attribute, Rule> => ({
  hdr: { mode: "any" },
  video_codec: { mode: "any" },
  audio_codec: { mode: "any" },
  channels: { mode: "any" },
  language: { mode: "any" },
});

const floors = (qualities: readonly TargetsQuality[]): Partial<Record<QualityKey, SizeDraft>> =>
  Object.fromEntries(
    qualities.map((quality) => [quality.key, { min: String(quality.floor), max: "" }]),
  );

/** A quality's typed size. Every quality the API lists has one. */
export function sizeOf(draft: TargetDraft, key: QualityKey): SizeDraft {
  const size = draft.sizes[key];
  if (size === undefined) {
    throw new Error(`no size for ${key}`);
  }
  return size;
}

/** A new Target: every quality but CAM accepted under the default preset, at their floors. */
export function newDraft({ qualities, presets }: TargetsQualityList): TargetDraft {
  const keys = qualities.map((quality) => quality.key);
  const accepted = new Set(keys.filter((key) => key !== cam));
  return {
    name: "",
    order: presetOrder(presets[0]?.tiers ?? [], accepted, keys),
    accepted,
    filters: anyRules(),
    sizes: floors(qualities),
  };
}

/** A saved Target as a draft, its qualities it doesn't accept below in the default preset's order. */
export function draftOf(
  target: TargetsTarget,
  { qualities, presets }: TargetsQualityList,
): TargetDraft {
  const keys = qualities.map((quality) => quality.key);
  const known = (key: string) => keys.find((candidate) => candidate === key);
  const tiers = target.tiers.map((tier) => (tier ?? []).flatMap((key) => known(key) ?? []));
  const cutoff = tiers.findIndex((tier) => tier.some((key) => key === target.cutoff));
  const rest = (presets[0]?.tiers ?? [])
    .flat()
    .flatMap((key) => (key === null ? [] : (known(key) ?? [])));
  const filters = anyRules();
  for (const filter of target.filters) {
    filters[filter.attribute] = { mode: filter.mode, values: filter.values };
  }
  const sizes = floors(qualities);
  for (const size of target.sizes) {
    sizes[size.quality] = { min: String(size.min), max: size.max === null ? "" : String(size.max) };
  }
  return {
    name: target.name,
    order: orderOf({ tiers, cutoff }, rest),
    accepted: new Set(tiers.flat()),
    filters,
    sizes,
  };
}

/** The draft with one quality accepted or not, the cutoff kept in range. */
export const accepting = (draft: TargetDraft, key: QualityKey, on: boolean): TargetDraft => {
  const accepted = new Set(draft.accepted);
  if (on) {
    accepted.add(key);
  } else {
    accepted.delete(key);
  }
  return { ...draft, accepted, order: keepCutoff(draft.order, accepted) };
};

export const layoutOfDraft = (draft: TargetDraft): Layout => layoutOf(draft.order, draft.accepted);

/** The accepted qualities, best first, as the Sizes section lists them. */
export const acceptedInOrder = (draft: TargetDraft): QualityKey[] =>
  layoutOfDraft(draft).tiers.flat();

const number = /^\d+(\.\d+)?$/u;

export type SizeProblem = { readonly min?: string; readonly max?: string };

/** What is wrong with one quality's typed size, if anything. */
export function sizeProblem({ min, max }: SizeDraft): SizeProblem | undefined {
  const minOk = number.test(min.trim());
  const maxOk = max.trim() === "" || number.test(max.trim());
  if (!minOk || !maxOk) {
    return {
      ...(!minOk && { min: "Enter a number of MiB per minute." }),
      ...(!maxOk && { max: "Enter a number, or leave it empty for no limit." }),
    };
  }
  if (max.trim() !== "" && Number(max) < Number(min)) {
    return { max: "The maximum is below the minimum." };
  }
  return undefined;
}

export const ruleProblem = (rule: Rule): string | undefined =>
  rule.mode !== "any" && rule.values.length === 0 ? "Pick at least one, or choose Any." : undefined;

/** Every reason the draft can't be saved, as sentences. */
export function problemsOf(draft: TargetDraft): string[] {
  const problems: string[] = [];
  if (draft.name.trim() === "") {
    problems.push("Name the Target.");
  }
  if (draft.accepted.size === 0) {
    problems.push("Accept at least one quality.");
  }
  if (attributes.some(({ key }) => ruleProblem(draft.filters[key]) !== undefined)) {
    problems.push("Pick a value for every filter that isn't Any.");
  }
  if (acceptedInOrder(draft).some((key) => sizeProblem(sizeOf(draft, key)) !== undefined)) {
    problems.push("Fix the sizes marked below.");
  }
  return problems;
}

/** The create body, once problemsOf finds nothing. */
export function bodyOf(draft: TargetDraft): TargetsNewTarget {
  const { tiers, cutoff } = layoutOfDraft(draft);
  const top = tiers[cutoff]?.[0];
  if (top === undefined) {
    throw new Error("a Target with no accepted quality has no body");
  }
  return {
    name: draft.name.trim(),
    tiers: tiers.map((tier) => [...tier]),
    cutoff: top,
    filters: attributes.flatMap(({ key }) => {
      const rule = draft.filters[key];
      return rule.mode === "any"
        ? []
        : [{ attribute: key, mode: rule.mode, values: [...rule.values] }];
    }),
    sizes: tiers.flat().map((quality) => {
      const { min, max } = sizeOf(draft, quality);
      return { quality, min: Number(min), max: max.trim() === "" ? null : Number(max) };
    }),
  };
}
