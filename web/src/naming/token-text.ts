import type { NamingToken, NamingTokenModifier } from "../api/gen/model";

/** Text that prints around a token's value only when the value isn't empty. */
export const wraps = [
  { value: "none", label: "None", prefix: "", suffix: "" },
  { value: "parentheses", label: "Parentheses", prefix: "(", suffix: ")" },
  { value: "brackets", label: "Square brackets", prefix: "[", suffix: "]" },
  { value: "braces", label: "Braces", prefix: "{", suffix: "}" },
  { value: "dash", label: "Dash before", prefix: "-", suffix: "" },
] as const;

export type WrapID = (typeof wraps)[number]["value"];

export const cases = [
  { value: "as-is", label: "As it is" },
  { value: "lower", label: "Lowercase" },
  { value: "upper", label: "Uppercase" },
] as const;

export type Case = (typeof cases)[number]["value"];

/** What may join a value's words in place of its spaces, as {Shelf.Title}. */
export const separators = [" ", ".", "-", "_"] as const;

export type Separator = (typeof separators)[number];

/** What the builder sets for one token: each control's value, whether the token offers it or not. */
export type TokenChoice = {
  readonly wrap: WrapID;
  /** A tag word such as "imdb", which wraps the token as {imdb-{ImdbId}}; "" for none. */
  readonly tag: string;
  readonly case: Case;
  readonly separator: Separator;
  /** The modifier's text: digits to pad or bytes to keep, languages, or Custom Format names. */
  readonly modifier: string;
  /** Keeps the end of a cut text, or leaves out the languages or Custom Formats listed. */
  readonly invert: boolean;
};

export const plainChoice: TokenChoice = {
  wrap: "none",
  tag: "",
  case: "as-is",
  separator: " ",
  modifier: "",
  invert: false,
};

/** Case and the separator change only words, so only titles and text take them. */
export const takesWords = (token: NamingToken): boolean =>
  token.kind === "title" || token.kind === "text";

const signed = (text: string, invert: boolean) => (invert ? `-${text}` : text);

/** What follows ":" for each modifier, from its text and whether it is inverted. */
const modifierTexts: Record<
  NonNullable<NamingTokenModifier>,
  (text: string, invert: boolean) => string
> = {
  pad: (text) => "0".repeat(Number(text)),
  cut: signed,
  languages: signed,
  formats: signed,
  format: (text) => text,
};

function nameText(token: NamingToken, choice: TokenChoice): string {
  if (!takesWords(token)) {
    return token.name;
  }
  const joined = token.name.replaceAll(" ", choice.separator);
  if (choice.case === "lower") {
    return joined.toLowerCase();
  }
  return choice.case === "upper" ? joined.toUpperCase() : joined;
}

/** The text the builder inserts for a token under the choices made. */
export function tokenText(token: NamingToken, choice: TokenChoice): string {
  const text = choice.modifier.trim();
  const modifier =
    token.modifier === null || text === ""
      ? ""
      : `:${modifierTexts[token.modifier](text, choice.invert)}`;
  const inner = nameText(token, choice) + modifier;
  if (choice.tag !== "") {
    return `{${choice.tag}-{${inner}}}`;
  }
  const wrap = wraps.find(({ value }) => value === choice.wrap) ?? wraps[0];
  return `{${wrap.prefix}${inner}${wrap.suffix}}`;
}
