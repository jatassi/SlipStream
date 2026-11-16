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
      text: "SlipStream keeps one file at each Target, whatever its cut. A file's cut only changes its name.",
      warning: false,
    };
  }
  const others = list.cuts.filter((cut) => cut.key !== standard).map((cut) => cut.name);
  if (others.length === list.cuts.length) {
    const them = others.length === 1 ? "it" : "them";
    return {
      text: `SlipStream searches for ${listed(others)} every time, even if no release of ${them} has turned up. It skips releases that name no cut, because they count as Standard.`,
      warning: true,
    };
  }
  if (others.length === 0) {
    return { text: "SlipStream always gets Standard.", warning: false };
  }
  return {
    text: `SlipStream always gets Standard. It also gets ${listed(others)} once a release or file of that cut turns up.`,
    warning: false,
  };
}
