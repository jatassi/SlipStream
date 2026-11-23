import { HttpResponse, http } from "msw";
import { expect, test } from "vitest";
import { page, userEvent } from "vitest/browser";

import { getListContentTypesMockHandler } from "../api/gen/query/works/works.msw";
import { problem } from "../test/handlers";
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
  const nav = page.getByRole("navigation", { name: "Settings" });
  await expect
    .element(nav.getByRole("link", { name: "Editions" }))
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
    .toMatchTextContent(/Black and Chrome can't be deleted.+Films and 1 file/u);
});
