import type { WorksContentType } from "../../api/gen/model";
import { releaseVerb } from "../monitoring/labels";
import { dayWords } from "../page/fields-format";

/** What folding reads of an Item row. */
export type Foldable = {
  readonly code: string;
  /** Its availability date; null while none is known */
  readonly date: string | null;
  readonly unreleased: boolean;
};

export type Fold<T> = { readonly items: readonly T[]; readonly words: string };

/** What folding reads of a Content Type: its Item noun and its verb for coming out. */
export type Folding = Pick<WorksContentType, "item" | "releases">;

export type Entry<T> =
  | { readonly kind: "item"; readonly item: T }
  | { readonly kind: "fold"; readonly fold: Fold<T> };

const dayMonth = new Intl.DateTimeFormat(undefined, {
  day: "numeric",
  month: "long",
  timeZone: "UTC",
});

const lastNumber = /^(.*?)(\d+)(\D*)$/u;

// Codes that differ only in their last number, such as S01E04 and S01E07, read as 4 and 7.
function shortCodes(first: string, last: string): [string, string] {
  const a = lastNumber.exec(first);
  const b = lastNumber.exec(last);
  if (a === null || b === null || a[1] !== b[1] || a[3] !== b[3]) {
    return [first, last];
  }
  return [String(Number(a[2])), String(Number(b[2]))];
}

const sameYear = (a: string, b: string) => a.slice(0, 4) === b.slice(0, 4);

function dateWords(date: string, short: boolean): string {
  const [year, month, day] = date.split("-").map(Number);
  if (!short || year === undefined || month === undefined || day === undefined) {
    return dayWords(date);
  }
  return dayMonth.format(new Date(Date.UTC(year, month - 1, day)));
}

function spanWords(first: string, last: string): string {
  const short = sameYear(first, last);
  return first === last
    ? dateWords(first, short)
    : `${dateWords(first, short)} to ${dateWords(last, short)}`;
}

/** "Episodes 4 to 7 air 12 October to 2 November", from the Item noun, the type's verb and the codes. */
export function foldWords(type: Folding, items: readonly Foldable[]): string {
  const first = items.at(0);
  const last = items.at(-1);
  if (first === undefined || last === undefined) {
    return "";
  }
  const [a, b] = shortCodes(first.code, last.code);
  const which = `${type.item.many} ${a} ${items.length === 2 ? "and" : "to"} ${b}`;
  return first.date === null || last.date === null
    ? `${which} have no availability date yet`
    : `${which} ${releaseVerb(type)} ${spanWords(first.date, last.date)}`;
}

const runOf = (item: Foldable) =>
  item.unreleased ? (item.date === null ? "undated" : "dated") : undefined;

/** The rows of a level: runs of two or more unreleased Items fold into one line, dated runs apart from undated ones. */
export function foldsOf<T extends Foldable>(type: Folding, items: readonly T[]): Entry<T>[] {
  const entries: Entry<T>[] = [];
  let run: T[] = [];
  const flush = () => {
    if (run.length >= 2) {
      entries.push({ kind: "fold", fold: { items: run, words: foldWords(type, run) } });
    } else {
      entries.push(...run.map((item) => ({ kind: "item" as const, item })));
    }
    run = [];
  };
  for (const item of items) {
    const kind = runOf(item);
    const at = run.at(0);
    if (at !== undefined && (kind === undefined || runOf(at) !== kind)) {
      flush();
    }
    if (kind === undefined) {
      entries.push({ kind: "item", item });
    } else {
      run.push(item);
    }
  }
  flush();
  return entries;
}
