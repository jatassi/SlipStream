import { expect, test } from "vitest";
import { page, userEvent } from "vitest/browser";

import { getListContentTypesMockHandler } from "../api/gen/query/works/works.msw";
import { changeOp, noImpact } from "../test/impact";
import { cinema } from "../test/library";
import { profileHandlers, ultra } from "../test/profiles";
import { renderApp } from "../test/render-app";
import { targetHandlers } from "../test/targets";
import { asPhone } from "../test/viewport";

function openEditor() {
  const op = changeOp("put", `*/api/v1/profiles/${ultra.id}`, () => noImpact, { profile: ultra });
  renderApp({
    path: `/settings/profiles/${ultra.id}`,
    handlers: [
      op.handler,
      ...targetHandlers(),
      ...profileHandlers(),
      getListContentTypesMockHandler({ contentTypes: [cinema] }),
    ],
  });
  return { op, section: page.getByRole("region", { name: "Subtitles" }) };
}

const languages = () => page.getByRole("list", { name: "Subtitle languages" });

async function add(name: string) {
  await userEvent.click(page.getByRole("button", { name: "Add language" }));
  await userEvent.click(page.getByRole("menuitem", { name, exact: true }));
}

async function choose(token: RegExp, option: string) {
  await userEvent.click(languages().getByRole("button", { name: token }).first());
  await userEvent.click(page.getByRole("menuitem", { name: option, exact: true }));
}

test("a new language reads as a sentence of menus, and each choice rewrites it and its meanings", async () => {
  openEditor();
  await add("English");
  const row = languages().getByRole("listitem");
  await expect
    .element(row.getByLabelText("Always get English subtitles, full, hearing impaired allowed."))
    .toBeVisible();
  await expect.element(row.getByText("Full: every line of dialogue.")).toBeVisible();

  await choose(/^When: /u, "Without English audio");
  await choose(/^Kind: /u, "forced only");
  await choose(/^Hearing impaired: /u, "avoided");
  await expect
    .element(
      row.getByLabelText(
        "Without English audio, get English subtitles, forced only, hearing impaired avoided.",
      ),
    )
    .toBeVisible();
  await expect
    .element(
      row.getByText(
        "Forced only: just the lines in another language, such as signs. Avoided: used only when nothing else fits.",
      ),
    )
    .toBeVisible();

  await choose(/^Language: /u, "Brazilian Portuguese");
  await choose(/^Hearing impaired: /u, "preferred");
  await expect
    .element(
      row.getByLabelText(
        "Without Brazilian Portuguese audio, get Brazilian Portuguese subtitles, forced only, hearing impaired preferred.",
      ),
    )
    .toBeVisible();
  await expect.element(row.getByText(/Preferred: chosen first when one fits\.$/u)).toBeVisible();
});

test("a language and kind listed twice is refused on the second, and Save sends nothing", async () => {
  const { op } = openEditor();
  await add("English");
  await add("English");
  await expect
    .element(languages().getByRole("alert"))
    .toMatchTextContent(/English subtitles are already/u);
  await userEvent.click(page.getByRole("button", { name: "Save", exact: true }));
  await expect
    .element(page.getByRole("alert").filter({ hasText: "Full English subtitles" }).last())
    .toBeVisible();
  expect(op.sent).toEqual([]);

  await choose(/^Kind: /u, "forced only");
  await expect.element(languages().getByRole("alert")).not.toBeInTheDocument();
  await userEvent.click(
    languages().getByRole("button", { name: "Remove English, full, hearing impaired allowed" }),
  );
  await expect.poll(() => languages().getByRole("listitem").all().length).toBe(1);
});

test("on a phone each part of the sentence opens an action sheet", async () => {
  await asPhone();
  openEditor();
  await userEvent.click(page.getByRole("button", { name: "Add language" }));
  await userEvent.click(page.getByRole("button", { name: "French", exact: true }));
  await userEvent.click(languages().getByRole("button", { name: "Kind: full" }));
  await expect.element(page.getByRole("dialog", { name: "Kind" })).toBeVisible();
});
