import { expect, test } from "vitest";

import { fileTokens } from "../test/naming";
import { pick, suggest } from "./suggest";

const names = (text: string, caret = text.length) =>
  suggest(fileTokens, text, caret)?.tokens.map((token) => token.name);

test("an open brace suggests every token, and what follows it narrows them by name or alias", () => {
  expect(names("{")).toHaveLength(fileTokens.length);
  expect(names("{Film Title} {(rel")).toEqual(["Year", "Release Group"]);
  expect(names("{edition")).toEqual(["Edition", "Edition Tag"]);
  expect(names("{film.ti")).toEqual(["Film Title"]);
  expect(names("{Year}")).toBeUndefined();
  expect(names("{season:0")).toBeUndefined();
  expect(names("{Nothing")).toBeUndefined();
});

test("a pick puts the token's name in place and closes it, keeping the caret before a typed prefix's suffix", () => {
  const plain = suggest(fileTokens, "{Film Title} - {tar", 19);
  const target = fileTokens.find((token) => token.name === "Target");
  if (plain === undefined || target === undefined) {
    throw new Error("no suggestions");
  }
  expect(pick("{Film Title} - {tar", plain, target)).toEqual({
    text: "{Film Title} - {Target}",
    caret: 23,
  });

  const year = fileTokens.find((token) => token.name === "Year");
  const wrapped = suggest(fileTokens, "{Film Title} {(ye", 17);
  if (wrapped === undefined || year === undefined) {
    throw new Error("no suggestions");
  }
  expect(pick("{Film Title} {(ye", wrapped, year)).toEqual({
    text: "{Film Title} {(Year}",
    caret: 19,
  });
});
