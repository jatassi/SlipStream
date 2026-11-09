import { expect, test } from "vitest";

import { fileTokens } from "../test/naming";
import { plainChoice, tokenText } from "./token-text";

const tokenNamed = (name: string) => {
  const found = fileTokens.find((token) => token.name === name);
  if (found === undefined) {
    throw new Error(`no token ${name}`);
  }
  return found;
};

test("the builder's text carries the wrap, case, separator, modifier or tag chosen", () => {
  expect(tokenText(tokenNamed("Year"), { ...plainChoice, wrap: "parentheses" })).toBe("{(Year)}");
  expect(tokenText(tokenNamed("Release Group"), { ...plainChoice, wrap: "dash" })).toBe(
    "{-Release Group}",
  );
  expect(
    tokenText(tokenNamed("Film Title"), { ...plainChoice, separator: ".", case: "lower" }),
  ).toBe("{film.title}");
  expect(
    tokenText(tokenNamed("Film Title"), { ...plainChoice, modifier: "30", invert: true }),
  ).toBe("{Film Title:-30}");
  expect(
    tokenText(tokenNamed("MediaInfo AudioLanguages"), {
      ...plainChoice,
      modifier: "EN+DE",
      wrap: "brackets",
    }),
  ).toBe("{[MediaInfo AudioLanguages:EN+DE]}");
  expect(
    tokenText(tokenNamed("ImdbId"), { ...plainChoice, tag: "imdb", wrap: "parentheses" }),
  ).toBe("{imdb-{ImdbId}}");
});
