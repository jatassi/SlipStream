import { HttpResponse, http } from "msw";
import { expect, test } from "vitest";
import { page, userEvent } from "vitest/browser";

import type { TargetsCut } from "../api/gen/model";
import { getListContentTypesMockHandler } from "../api/gen/query/works/works.msw";
import { problem } from "../test/handlers";
import { changeOp, noImpact } from "../test/impact";
import { cinema } from "../test/library";
import { blackAndChrome, profileHandlers } from "../test/profiles";
import { renderApp } from "../test/render-app";

const handlers = () => [
  ...profileHandlers(),
  getListContentTypesMockHandler({ contentTypes: [cinema] }),
];

const table = () => page.getByRole("table", { name: "Cuts" });
const row = (name: string) => table().getByRole("row", { name: new RegExp(`^${name}`, "u") });

test("Editions lists every cut with what titles name it by and where it comes from", async () => {
  renderApp({ path: "/settings/editions", handlers: handlers() });
  const tabs = page.getByRole("navigation", { name: "Settings" });
  await expect
    .element(tabs.getByRole("link", { name: "Editions" }))
    .toHaveAttribute("aria-current", "page");
  await expect
    .element(row("Standard"))
    .toHaveTextContent("StandardAny title that names no cutBuilt in");
  await expect.element(row("Extended")).toHaveTextContent("ExtendedExtended, Extended CutBuilt in");
  await expect.element(row("Superfan")).toHaveTextContent("SuperfanSuperfanFilms");
  await expect
    .element(row("Black and Chrome"))
    .toHaveTextContent("Black and ChromeBlack and Chrome, B&CCustom");
  expect(
    table()
      .getByRole("button", { name: /^Actions for /u })
      .all(),
  ).toHaveLength(1);
  await expect
    .element(page.getByRole("button", { name: "Actions for Black and Chrome" }))
    .toBeVisible();
});

test("Add cut makes a custom cut from its name and phrases split by commas", async () => {
  const sent: unknown[] = [];
  renderApp({
    path: "/settings/editions",
    handlers: [
      ...handlers(),
      http.post("*/api/v1/cuts", async ({ request }) => {
        sent.push(await request.json());
        const made: TargetsCut = {
          key: "superfan-cut",
          name: "Superfan Cut",
          aliases: ["Superfan Cut", "SFC", "Fan Cut"],
          origin: "custom",
        };
        return HttpResponse.json(made, { status: 201 });
      }),
    ],
  });
  await userEvent.click(page.getByRole("button", { name: "Add cut" }));
  const dialog = page.getByRole("dialog", { name: "Add a cut" });
  await userEvent.click(dialog.getByRole("button", { name: "Add cut" }));
  await expect.element(dialog.getByText("Name the cut.")).toBeVisible();
  expect(sent).toEqual([]);

  await userEvent.fill(dialog.getByRole("textbox", { name: "Name" }), " Superfan Cut ");
  await userEvent.fill(dialog.getByRole("textbox", { name: "Also read as" }), "SFC, , Fan Cut ");
  await userEvent.click(dialog.getByRole("button", { name: "Add cut" }));
  await expect
    .element(page.getByText("Added Superfan Cut. Profiles and Works can list it now."))
    .toBeVisible();
  await expect.element(dialog).not.toBeInTheDocument();
  expect(sent).toEqual([{ name: "Superfan Cut", aliases: ["SFC", "Fan Cut"] }]);
});

test("a name another cut goes by is refused on the name", async () => {
  renderApp({
    path: "/settings/editions",
    handlers: [
      ...handlers(),
      http.post("*/api/v1/cuts", () =>
        problem(409, "cut_name_taken", "A cut already has that name, alias or key"),
      ),
    ],
  });
  await userEvent.click(page.getByRole("button", { name: "Add cut" }));
  const dialog = page.getByRole("dialog", { name: "Add a cut" });
  await userEvent.fill(dialog.getByRole("textbox", { name: "Name" }), "Extended");
  await userEvent.click(dialog.getByRole("button", { name: "Add cut" }));
  await expect
    .element(dialog.getByText("Another cut already uses this name or one of these phrases."))
    .toBeVisible();
});

test("renaming a custom cut shows the files it renames, then saves through the Impact", async () => {
  const op = changeOp(
    "put",
    `*/api/v1/cuts/${blackAndChrome.key}`,
    () => ({ ...noImpact, works: 1, renamed: 2 }),
    { cut: { ...blackAndChrome, name: "Noir" } },
  );
  renderApp({ path: "/settings/editions", handlers: [op.handler, ...handlers()] });
  await userEvent.click(page.getByRole("button", { name: "Actions for Black and Chrome" }));
  await userEvent.click(page.getByRole("menuitem", { name: "Edit" }));
  const dialog = page.getByRole("dialog", { name: "Edit Black and Chrome" });
  await expect
    .element(dialog.getByRole("textbox", { name: "Name" }))
    .toHaveValue("Black and Chrome");
  await expect.element(dialog.getByRole("textbox", { name: "Also read as" })).toHaveValue("B&C");
  await userEvent.fill(dialog.getByRole("textbox", { name: "Name" }), "Noir");
  await userEvent.click(dialog.getByRole("button", { name: "Save" }));

  const sheet = page.getByRole("dialog", { name: "Save Black and Chrome" });
  await expect
    .element(sheet.getByRole("list", { name: "What changes" }))
    .toHaveTextContent("1 Work changes.2 files are renamed.");
  expect(sheet.getByRole("radiogroup").query()).toBeNull();
  await userEvent.click(sheet.getByRole("button", { name: "Save", exact: true }));

  await expect.element(page.getByText("Saved Noir.")).toBeVisible();
  await expect.element(dialog).not.toBeInTheDocument();
  const body = { name: "Noir", aliases: ["B&C"] };
  expect(op.sent).toEqual([
    { query: { preview: "true" }, body },
    { query: { unwanted: "delete", confirm: "0" }, body },
  ]);
});

test("deleting a cut a Profile lists is refused with what holds it", async () => {
  renderApp({
    path: "/settings/editions",
    handlers: [
      ...handlers(),
      http.delete(`*/api/v1/cuts/${blackAndChrome.key}`, () =>
        HttpResponse.json(
          {
            status: 409,
            code: "cut_in_use",
            title: "Edition lists or files hold the cut",
            detail: "Films and 1 file use it",
          },
          { status: 409, headers: { "Content-Type": "application/problem+json" } },
        ),
      ),
    ],
  });
  await userEvent.click(page.getByRole("button", { name: "Actions for Black and Chrome" }));
  await userEvent.click(page.getByRole("menuitem", { name: "Delete" }));
  const dialog = page.getByRole("dialog", { name: "Delete Black and Chrome?" });
  await userEvent.click(dialog.getByRole("button", { name: "Delete" }));
  await expect
    .element(dialog.getByRole("alert"))
    .toHaveTextContent("Black and Chrome can't be deleted. Films and 1 file use it.");
});
