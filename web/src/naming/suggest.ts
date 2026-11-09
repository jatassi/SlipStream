import type { NamingToken } from "../api/gen/model";

/** The tokens that fit what is typed after an open brace, and the text a pick replaces. */
export type Suggestions = {
  readonly tokens: readonly NamingToken[];
  /** Where the typed name starts, after the brace and any prefix such as "(". */
  readonly from: number;
  readonly to: number;
  /** A prefix was typed, so the caret stays before "}" for its suffix. */
  readonly prefixed: boolean;
};

const prefixes = new Set([" ", "-", ".", "_", "[", "("]);

/** A name as the grammar matches it: no case, spaces, dots, dashes or underscores. */
export const keyOf = (name: string): string => name.toLowerCase().replaceAll(/[\s._-]/gu, "");

/** The suggestions while the caret sits in a token that isn't closed, starting as "{" is typed. */
export function suggest(
  tokens: readonly NamingToken[],
  text: string,
  caret: number,
): Suggestions | undefined {
  const open = text.lastIndexOf("{", caret - 1);
  if (open === -1 || /[}:]/u.test(text.slice(open, caret))) {
    return undefined;
  }
  let from = open + 1;
  while (from < caret && prefixes.has(text.charAt(from))) {
    from += 1;
  }
  const typed = keyOf(text.slice(from, caret));
  const keys = (token: NamingToken) => [token.name, ...token.aliases].map((name) => keyOf(name));
  const starting = tokens.filter((token) => keys(token).some((key) => key.startsWith(typed)));
  const within = tokens.filter(
    (token) => !starting.includes(token) && keys(token).some((key) => key.includes(typed)),
  );
  const found = [...starting, ...within];
  return found.length === 0
    ? undefined
    : { tokens: found, from, to: caret, prefixed: from > open + 1 };
}

/** The text with a picked token in place of what was typed, and the caret after it. */
export function pick(
  text: string,
  { from, to, prefixed }: Suggestions,
  token: NamingToken,
): { text: string; caret: number } {
  const closed = text.charAt(to) === "}";
  const inserted = closed ? token.name : `${token.name}}`;
  const caret = from + token.name.length + (prefixed ? 0 : 1);
  return { text: text.slice(0, from) + inserted + text.slice(to), caret };
}
