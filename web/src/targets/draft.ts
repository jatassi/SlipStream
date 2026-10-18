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
  type PresetTiers,
  accept,
  cam,
  drop,
  layoutOf,
  orderOf,
  presetOrder,
  rankedOf,
} from "./order";
import { type Attribute, attributes } from "./vocab";

export type Rule =
  | { readonly mode: "any" }
  | { readonly mode: TargetsFilterMode; readonly values: readonly string[] };

/** A size as typed, in MiB per minute; an empty maximum is no limit. */
export type SizeDraft = { readonly min: string; readonly max: string };

/**
 * A Target being edited. The order ranks the accepted qualities but CAM,
 * and `cinema` says whether CAM is accepted; sizes are kept for every
 * quality and only the accepted ones are sent.
 */
export type TargetDraft = {
  readonly name: string;
  readonly order: Order;
  readonly cinema: boolean;
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

export function sizeOf(draft: TargetDraft, key: QualityKey): SizeDraft {
  const size = draft.sizes[key];
  if (size === undefined) {
    throw new Error(`no size for ${key}`);
  }
  return size;
}

/** A new Target: every quality but CAM accepted under the default preset, at their floors. */
export function newDraft({ qualities, presets }: TargetsQualityList): TargetDraft {
  return {
    name: "",
    order: presetOrder(
      presets[0]?.tiers ?? [],
      qualities.map((quality) => quality.key),
    ),
    cinema: false,
    filters: anyRules(),
    sizes: floors(qualities),
  };
}

export function draftOf(target: TargetsTarget, { qualities }: TargetsQualityList): TargetDraft {
  const keys = qualities.map((quality) => quality.key);
  const known = (key: string) => keys.find((candidate) => candidate === key);
  const tiers = target.tiers.map((tier) => (tier ?? []).flatMap((key) => known(key) ?? []));
  const cutoff = tiers.findIndex((tier) => tier.some((key) => key === target.cutoff));
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
    order: orderOf({ tiers, cutoff }),
    cinema: tiers.some((tier) => tier.includes(cam)),
    filters,
    sizes,
  };
}

export const accepting = (
  draft: TargetDraft,
  key: QualityKey,
  on: boolean,
  preset: PresetTiers,
): TargetDraft => ({
  ...draft,
  order: on ? accept(draft.order, key, preset) : drop(draft.order, key),
});

/** The qualities the draft doesn't accept, CAM left out, best first. */
export const unacceptedOf = (draft: TargetDraft, keys: readonly QualityKey[]): QualityKey[] => {
  const ranked = new Set(rankedOf(draft.order));
  return keys.filter((key) => key !== cam && !ranked.has(key)).toReversed();
};

export const layoutOfDraft = (draft: TargetDraft): Layout => layoutOf(draft.order, draft.cinema);

export const acceptedInOrder = (draft: TargetDraft): QualityKey[] =>
  layoutOfDraft(draft).tiers.flat();

const number = /^\d+(\.\d+)?$/u;

type SizeProblem = { readonly min?: string; readonly max?: string };

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

export function problemsOf(draft: TargetDraft): string[] {
  const problems: string[] = [];
  if (draft.name.trim() === "") {
    problems.push("Name the Target.");
  }
  if (layoutOfDraft(draft).tiers.length === 0) {
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
