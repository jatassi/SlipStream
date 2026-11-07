import { expect, test } from "vitest";
import { page, userEvent } from "vitest/browser";

import { breadcrumbs } from "../../shell/titles";
import { addIndexerServer } from "../../test/add-indexer-server";
import { renderApp } from "../../test/render-app";

const customPage = "/settings/indexers/new/custom";
const definition = () => page.getByRole("textbox", { name: "Definition" });
const add = () => page.getByRole("button", { name: "Add definition" });

test("a pasted definition is added and opens its form", async () => {
  const server = addIndexerServer();
  const { router } = renderApp({ path: customPage, handlers: server.handlers });
  await expect
    .element(page.getByRole("heading", { name: "Add a custom definition" }))
    .toBeVisible();
  expect(breadcrumbs(router.state.matches).map((crumb) => crumb.title)).toEqual([
    "Settings",
    "Indexers",
    "Add Indexer",
    "Add a custom definition",
  ]);
  await userEvent.click(add());
  await expect.element(page.getByText("Paste the definition, or choose its file.")).toBeVisible();
  await userEvent.fill(definition(), "id: meadow\nname: Meadow\n");
  await userEvent.click(add());
  await expect.element(page.getByRole("heading", { name: "Add Meadow", level: 1 })).toBeVisible();
  expect(router.state.location.pathname).toBe("/settings/indexers/new/catalog/meadow");
  expect(breadcrumbs(router.state.matches).map((crumb) => crumb.title)).toEqual([
    "Settings",
    "Indexers",
    "Add Indexer",
    "Add Meadow",
  ]);
  expect(server.writes).toContainEqual({
    method: "POST",
    path: "/indexer-catalog/custom",
    body: { yaml: "id: meadow\nname: Meadow\n" },
  });
});

test("a chosen file fills the definition", async () => {
  renderApp({ path: customPage, handlers: addIndexerServer().handlers });
  const file = new File(["id: meadow\nname: Meadow\n"], "meadow.yml", { type: "application/yaml" });
  await userEvent.upload(page.getByLabelText("Definition file"), file);
  await expect.element(definition()).toHaveValue("id: meadow\nname: Meadow\n");
  await expect.element(page.getByText("meadow.yml")).toBeVisible();
});

test("a definition SlipStream can't run says so, with the reason under Details", async () => {
  renderApp({ path: customPage, handlers: addIndexerServer().handlers });
  await userEvent.fill(definition(), "id: bramble\nlogin:\n  method: oauth\n");
  await userEvent.click(add());
  await expect
    .element(page.getByRole("alert").getByText("SlipStream can't run this definition yet."))
    .toBeVisible();
  await userEvent.click(page.getByRole("button", { name: "Details" }));
  await expect
    .element(
      page.getByText(
        "The definition signs in with the method 'oauth', which SlipStream doesn't run.",
      ),
    )
    .toBeVisible();
});

test("a definition whose id the catalog has is refused", async () => {
  renderApp({ path: customPage, handlers: addIndexerServer().handlers });
  await userEvent.fill(definition(), "id: marsh\nname: Marsh\n");
  await userEvent.click(add());
  await expect
    .element(
      page
        .getByRole("alert")
        .getByText("The Indexer catalog already has a definition with this id."),
    )
    .toBeVisible();
});
