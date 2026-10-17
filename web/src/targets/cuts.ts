import type {
  TargetsCut,
  TargetsCutOrigin,
  TargetsNewCut,
  WorksContentType,
} from "../api/gen/model";

const origins: Record<
  TargetsCutOrigin,
  (cut: TargetsCut, types: readonly WorksContentType[]) => string
> = {
  builtin: () => "Built in",
  type: (cut, types) => types.find((type) => type.id === cut.contentType)?.name ?? "",
  custom: () => "Custom",
};

/** Where a cut comes from, in words: built in, the Content Type that adds it, or custom. */
export const originOf = (cut: TargetsCut, types: readonly WorksContentType[]): string =>
  origins[cut.origin](cut, types);

export const isCustom = (cut: TargetsCut): boolean => cut.origin === "custom";

/** The phrases a title names the cut by. Standard has none: it is every title that names no cut. */
export const readAsOf = (cut: TargetsCut): string =>
  cut.aliases.length === 0 ? "Any title that names no cut" : cut.aliases.join(", ");

/** A custom cut being written: its name, and its other phrases split by commas. */
export type CutDraft = { readonly name: string; readonly aliases: string };

export const newCutDraft: CutDraft = { name: "", aliases: "" };

// A custom cut's aliases start with its name, which the draft holds apart.
export const cutDraftOf = (cut: TargetsCut): CutDraft => ({
  name: cut.name,
  aliases: cut.aliases.slice(1).join(", "),
});

export const cutBodyOf = (draft: CutDraft): TargetsNewCut => ({
  name: draft.name.trim(),
  aliases: draft.aliases
    .split(",")
    .map((alias) => alias.trim())
    .filter((alias) => alias !== ""),
});
