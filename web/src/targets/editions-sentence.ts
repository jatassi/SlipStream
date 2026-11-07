import { listed } from "./words";

export const standard = "standard";

export type Cut = { readonly key: string; readonly name: string };

export type EditionList =
  | { readonly kind: "any" }
  | { readonly kind: "cuts"; readonly cuts: readonly [Cut, ...Cut[]] };

export type EditionsSentence = { readonly text: string; readonly warning: boolean };

export function editionsSentence(list: EditionList): EditionsSentence {
  if (list.kind === "any") {
    return {
      text: "Each Item wants one file at every Target, of whatever cut. A file's cut only sets its name.",
      warning: false,
    };
  }
  const others = list.cuts.filter((cut) => cut.key !== standard).map((cut) => cut.name);
  if (others.length === list.cuts.length) {
    const them = others.length === 1 ? "it" : "them";
    return {
      text: `Every Item wants ${listed(others)} outright, even where no Release of ${them} has been seen. A Release that names no cut is Standard, which this list doesn't want.`,
      warning: true,
    };
  }
  if (others.length === 0) {
    return { text: "Every Item always wants Standard.", warning: false };
  }
  return {
    text: `Every Item always wants Standard. It wants ${listed(others)} only once a Release or file of that cut has been seen for it.`,
    warning: false,
  };
}
