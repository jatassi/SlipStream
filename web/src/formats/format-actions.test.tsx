import { expect, test } from "vitest";
import { page, userEvent } from "vitest/browser";

import type { FormatsCustomFormat, FormatsSchema } from "../api/gen/model";
import {
  getCustomFormatSchemaMockHandler,
  getDuplicateCustomFormatMockHandler,
  getGetCustomFormatMockHandler,
  getListCustomFormatsMockHandler,
} from "../api/gen/query/formats/formats.msw";
import { changeOp, noImpact } from "../test/impact";
import { renderApp } from "../test/render-app";

const schema: FormatsSchema = {
  dialects: (["slipstream", "radarr", "sonarr"] as const).map((dialect) => ({
    dialect,
    kinds: [
      {
        kind: "ReleaseTitleSpecification",
        words: "Release title",
        fields: ["value"],
        pattern: true,
        values: [],
      },
    ],
  })),
};

const golden: FormatsCustomFormat = {
  id: "0199e1f2-7a3b-7c4d-8e5f-0000000000f1",
  name: "Golden",
  includeWhenRenaming: false,
  dialect: "radarr",
  conditions: [
    {
      name: "Golden",
      kind: "ReleaseTitleSpecification",
      negate: false,
      required: false,
      fields: { value: "GOLDEN" },
    },
  ],
  scoredBy: [
    { id: "0199e1f2-7a3b-7c4d-8e5f-0000000000a1", name: "1080p" },
    { id: "0199e1f2-7a3b-7c4d-8e5f-0000000000a2", name: "2160p" },
  ],
  slowReleases: 3,
};

const handlers = [
  getListCustomFormatsMockHandler({ formats: [golden], matchLimitMilliseconds: 100 }),
  getCustomFormatSchemaMockHandler(schema),
  getGetCustomFormatMockHandler(golden),
];

const twoWorks = { ...noImpact, works: 2 };

test("deleting a scored format names its Targets, then shows the Impact before it deletes", async () => {
  const op = changeOp("delete", `*/api/v1/custom-formats/${golden.id}`, () => twoWorks, {
    scoredBy: golden.scoredBy,
  });
  renderApp({ path: "/settings/custom-formats", handlers: [op.handler, ...handlers] });
  await userEvent.click(page.getByRole("button", { name: "Actions for Golden" }));
  await userEvent.click(page.getByRole("menuitem", { name: "Delete" }));
  const asking = page.getByRole("dialog", { name: "Delete Golden?" });
  await expect
    .element(asking.getByText("1080p and 2160p score it. Deleting it also deletes those scores."))
    .toBeVisible();
  await userEvent.click(asking.getByRole("button", { name: "Delete Golden" }));

  const impact = page.getByRole("dialog", { name: "Delete Golden", exact: true });
  await expect
    .element(impact.getByRole("list", { name: "What changes" }))
    .toHaveTextContent("2 Works change.");
  await userEvent.click(impact.getByRole("button", { name: "Delete Golden" }));
  await expect.element(page.getByText("Deleted Golden.")).toBeVisible();
  expect(op.sent.map((call) => call.query)).toEqual([
    { preview: "true" },
    { unwanted: "delete", confirm: "0" },
  ]);
});

test("Duplicate copies a format and says what it made", async () => {
  const copied: string[] = [];
  renderApp({
    path: "/settings/custom-formats",
    handlers: [
      getDuplicateCustomFormatMockHandler(({ params }) => {
        copied.push(String(params["formatId"]));
        return {
          ...golden,
          id: "0199e1f2-7a3b-7c4d-8e5f-0000000000f2",
          name: "Golden (2)",
          scoredBy: [],
        };
      }),
      ...handlers,
    ],
  });
  await userEvent.click(page.getByRole("button", { name: "Actions for Golden" }));
  await userEvent.click(page.getByRole("menuitem", { name: "Duplicate" }));
  await expect.element(page.getByText("Made Golden (2) from Golden.")).toBeVisible();
  expect(copied).toEqual([golden.id]);
});

test("a saved format's editor warns of its slow pattern and saves its edit through the Impact", async () => {
  const op = changeOp("put", `*/api/v1/custom-formats/${golden.id}`, () => twoWorks, {
    format: golden,
  });
  renderApp({ path: `/settings/custom-formats/${golden.id}`, handlers: [op.handler, ...handlers] });
  await expect
    .element(
      page.getByText(
        "A pattern ran past its 100 ms limit on 3 Releases since this Custom Format was saved. Those Releases count as not matching it. Shorten or simplify the pattern.",
      ),
    )
    .toBeVisible();
  await expect
    .element(page.getByText("Pasted from Radarr, so its values are Radarr's."))
    .toBeVisible();
  await userEvent.fill(page.getByRole("textbox", { name: "Pattern" }), String.raw`\bGOLDEN\b`);
  await userEvent.click(page.getByRole("button", { name: "Save" }));

  const impact = page.getByRole("dialog", { name: "Save Golden" });
  await expect
    .element(impact.getByRole("list", { name: "What changes" }))
    .toHaveTextContent("2 Works change.");
  await userEvent.click(impact.getByRole("button", { name: "Save" }));
  await expect.element(page.getByText("Saved Golden.")).toBeVisible();
  expect(op.sent.map((call) => call.query)).toEqual([
    { preview: "true" },
    { unwanted: "delete", confirm: "0" },
  ]);
  expect(op.sent[1]?.body).toEqual({
    name: "Golden",
    includeWhenRenaming: false,
    conditions: [{ ...golden.conditions[0], fields: { value: String.raw`\bGOLDEN\b` } }],
  });
});
