import type { FormatsPasted, FormatsPastedApp, FormatsPastedFormat } from "../api/gen/model";

export type App = FormatsPastedApp;

export const appWords: Readonly<Record<App, string>> = { radarr: "Radarr", sonarr: "Sonarr" };

/**
 * What a paste replaces at first: each Custom Format of a pasted one's
 * name that holds the same app's values. One of the other app's keeps
 * beside it, because a Radarr and a Sonarr format of one name are two
 * formats.
 */
export const replacedAtFirst = ({ app, formats }: FormatsPasted): string[] =>
  formats.flatMap(({ sameName }) => (sameName?.dialect === app ? [sameName.id] : []));

export function outcomeWords({ outcome, savedAs, sameName }: FormatsPastedFormat): string {
  switch (outcome) {
    case "add":
      return "Adds it.";
    case "replace":
      return `Replaces the ${sameName?.name ?? savedAs} you have.`;
    case "keep_both":
      return `Adds it as ${savedAs}, beside the ${sameName?.name ?? ""} you have.`;
    default:
      return outcome satisfies never;
  }
}

const many = (count: number) => (count === 1 ? "Custom Format" : "Custom Formats");

/** The paste's save button, such as "Add 3 Custom Formats" or "Replace 1 and add 2". */
export function pasteLabel(formats: readonly FormatsPastedFormat[]): string {
  const replaced = formats.filter((format) => format.outcome === "replace").length;
  const added = formats.length - replaced;
  if (replaced === 0) {
    return `Add ${String(added)} ${many(added)}`;
  }
  if (added === 0) {
    return `Replace ${String(replaced)} ${many(replaced)}`;
  }
  return `Replace ${String(replaced)} and add ${String(added)}`;
}
