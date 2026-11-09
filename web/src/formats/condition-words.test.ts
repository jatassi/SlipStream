import { expect, test } from "vitest";

import type { FormatsCustomFormat } from "../api/gen/model";
import { matchSentence, slowSentence } from "./condition-words";
import { type Kind, bodyOf, conditionsOf } from "./draft";

const kinds: readonly Kind[] = [
  {
    kind: "ReleaseTitleSpecification",
    words: "Release title",
    fields: ["value"],
    pattern: true,
    values: [],
  },
  {
    kind: "LanguageSpecification",
    words: "Language",
    fields: ["value", "exceptLanguage"],
    pattern: false,
    values: [
      { value: 1, words: "English" },
      { value: 2, words: "French" },
    ],
  },
  {
    kind: "SourceSpecification",
    words: "Source",
    fields: ["value"],
    pattern: false,
    values: [
      { value: 7, words: "WEB-DL" },
      { value: 9, words: "Bluray" },
    ],
  },
  { kind: "SizeSpecification", words: "Size", fields: ["min", "max"], pattern: false, values: [] },
  { kind: "YearSpecification", words: "Year", fields: ["min", "max"], pattern: false, values: [] },
];

type Condition = FormatsCustomFormat["conditions"][number];

const title = (value: string, more: Partial<Condition> = {}): Condition => ({
  name: value,
  kind: "ReleaseTitleSpecification",
  negate: false,
  required: false,
  fields: { value },
  ...more,
});

const of = (kind: string, fields: Condition["fields"], more: Partial<Condition> = {}) => ({
  name: kind,
  kind,
  negate: false,
  required: false,
  fields,
  ...more,
});

const sentence = (conditions: Condition[]) => matchSentence(conditionsOf({ conditions }, kinds));

test.each<[string, Condition[], string | undefined]>([
  ["no conditions", [], undefined],
  ["a blank pattern", [title("")], undefined],
  [
    "one pattern",
    [title(String.raw`\bDV\b`)],
    String.raw`Matches a Release when its release title matches "\bDV\b".`,
  ],
  [
    "two patterns of one kind, either of which matches",
    [title("DV"), title("DoVi")],
    'Matches a Release when its release title matches "DV" or its release title matches "DoVi".',
  ],
  [
    "three patterns of one kind",
    [title("a"), title("b"), title("c")],
    'Matches a Release when its release title matches "a", its release title matches "b", or its release title matches "c".',
  ],
  [
    "required conditions, which every one must match and which leave the others out",
    [
      title("x265", { required: true }),
      title("HEVC"),
      title("10bit", { required: true, negate: true }),
    ],
    `Matches a Release when its release title matches "x265" and its release title doesn't match "10bit".`,
  ],
  [
    "two kinds, both of which must match",
    [title("DV"), of("SourceSpecification", { value: 9 }, { negate: true })],
    `Matches a Release when its release title matches "DV", and its source isn't Bluray.`,
  ],
  [
    "a language and any language but one",
    [
      of("LanguageSpecification", { value: 2, exceptLanguage: false }),
      of("LanguageSpecification", { value: 1, exceptLanguage: true }),
    ],
    "Matches a Release when its language is French or it has a language other than English.",
  ],
  [
    "only one language",
    [of("LanguageSpecification", { value: 1, exceptLanguage: true }, { negate: true })],
    "Matches a Release when its only language is English.",
  ],
  [
    "a size and a year",
    [
      of("SizeSpecification", { min: 1, max: 10 }),
      of("YearSpecification", { min: 1990, max: 1999 }),
    ],
    "Matches a Release when its size is over 1 GB and at most 10 GB, and its year is 1990 to 1999.",
  ],
])("%s", (_, conditions, want) => {
  expect(sentence(conditions)).toBe(want);
});

test("a format reads back into the body it was read from, naming an unnamed condition by its value", () => {
  const conditions = [
    title(String.raw`\bDV\b`),
    of("LanguageSpecification", { value: 1, exceptLanguage: true }),
    of("SizeSpecification", { min: 1.5, max: 10 }),
    { ...title("HDR10"), name: "" },
  ];
  const body = bodyOf({
    name: " HDR ",
    includeWhenRenaming: true,
    dialect: "radarr",
    conditions: conditionsOf({ conditions }, kinds),
  });
  expect(body).toEqual({
    name: "HDR",
    includeWhenRenaming: true,
    conditions: [...conditions.slice(0, 3), { ...title("HDR10"), name: "HDR10" }],
  });
});

test.each<[number, string]>([
  [
    1,
    "A pattern ran past its 100 ms limit on 1 Release since this Custom Format was saved. That Release counts as not matching it. Shorten or simplify the pattern.",
  ],
  [
    3,
    "A pattern ran past its 100 ms limit on 3 Releases since this Custom Format was saved. Those Releases count as not matching it. Shorten or simplify the pattern.",
  ],
])("the slow-pattern warning for %i Releases", (releases, want) => {
  expect(slowSentence(releases, 100)).toBe(want);
});
