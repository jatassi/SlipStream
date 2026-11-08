import { type ApiError, fieldErrors } from "../api/errors";
import type {
  FormatsEvaluated,
  FormatsExplained,
  FormatsParseKind,
  FormatsParsed,
  FormatsReadFacts,
  FormatsRevision,
} from "../api/gen/model";
import type { Problems } from "./draft";

export type TesterKind = FormatsParseKind;

export const nameLimit = 100;

export const kindLabels = {
  release: "Release names",
  file: "File paths",
} as const satisfies Record<TesterKind, string>;

/** Each non-blank line of text, trimmed. */
export const namesIn = (text: string): string[] =>
  text
    .split(/\r?\n/u)
    .map((line) => line.trim())
    .filter((line) => line !== "");

export function namesProblem(names: readonly string[], kind: TesterKind): string | undefined {
  if (names.length <= nameLimit) {
    return undefined;
  }
  const noun = kind === "file" ? "paths" : "names";
  return `This has ${String(names.length)} ${noun}. Test ${String(nameLimit)} or fewer at a time.`;
}

function revisionWords({ version, real, repack }: FormatsRevision): string | undefined {
  const parts = [
    ...(version > 1 ? [`${repack ? "Repack" : "Proper"}, version ${String(version)}`] : []),
    ...(real > 0 ? [`Real ${String(real)}`] : []),
  ];
  return parts.length === 0 ? undefined : parts.join(", ");
}

export type FactRow = { readonly label: string; readonly value: string };

/** What a name reads as, one row per fact the name gives. */
export function factRows(facts: FormatsReadFacts): FactRow[] {
  const ids = Object.entries(facts.ids).map(([tag, id]) => `${tag} ${id}`);
  const rows: [string, string | null | undefined][] = [
    ["Titles", facts.titles.join(", ")],
    ["Year", facts.year === null ? null : String(facts.year)],
    ["Numbering", facts.marks.join(", ")],
    ["Quality", facts.quality],
    ["Source", facts.source],
    ["Resolution", facts.resolution],
    ["Modifier", facts.modifier],
    ["Revision", revisionWords(facts.revision)],
    ["Release group", facts.group],
    ["Edition", facts.edition],
    ["Cut", facts.cut === "standard" ? null : facts.cut],
    ["HDR", facts.hdr.join(", ")],
    ["Video codec", facts.video],
    ["Audio", facts.audio.join(", ")],
    ["Channels", facts.channels.join(", ")],
    ["Languages", facts.languages.join(", ")],
    ["Id tags", ids.join(", ")],
  ];
  return rows.flatMap(([label, value]) =>
    value === null || value === undefined || value === "" ? [] : [{ label, value }],
  );
}

/**
 * The formats whose conditions the tester lists, the matched ones and the
 * draft, and the names of the rest, which a TRaSH library makes many of.
 */
export function formatGroups(formats: readonly FormatsExplained[]): {
  shown: FormatsExplained[];
  others: string[];
} {
  const shown = formats.filter((format) => format.matched || format.draft);
  const others = formats
    .filter((format) => !format.matched && !format.draft)
    .map((format) => format.name);
  return { shown, others };
}

export function formatResult(format: FormatsExplained): string {
  if (format.timedOut) {
    return "Doesn't match, because a pattern ran past its time limit";
  }
  return format.matched ? "Matches" : "Doesn't match";
}

export function conditionResult(condition: FormatsEvaluated): string {
  if (condition.timedOut) {
    return "Fails, because its pattern ran past its time limit";
  }
  return condition.matched ? "Passes" : "Fails";
}

export const conditionNotes = (condition: FormatsEvaluated): string[] => [
  ...(condition.negate ? ["Negate"] : []),
  ...(condition.required ? ["Required"] : []),
];

/**
 * A name judged against one Work fills its cells. One that matches several
 * Works or none is judged at every Target, which accepts it or doesn't.
 */
export function verdictWords(result: FormatsParsed, wouldFill: boolean): string {
  const oneWork = result.works.length === 1;
  if (wouldFill) {
    return oneWork ? "Would fill" : "Accepts it";
  }
  return oneWork ? "Wouldn't fill" : "Doesn't accept it";
}

/**
 * A draft's refusal keyed as the editor's own problems are: the API names
 * the field under the body's draft.
 */
export const draftProblems = (error: ApiError): Problems =>
  Object.fromEntries(
    Object.entries(fieldErrors(error)).map(([at, text]) => [at.replace(/^draft\./u, ""), text]),
  );
