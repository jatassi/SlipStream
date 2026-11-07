import { HttpResponse, http } from "msw";
import { expect, test } from "vitest";
import { page, userEvent } from "vitest/browser";

import { addIndexerServer } from "../../test/add-indexer-server";
import { notDownloaded } from "../../test/catalog";
import { renderApp } from "../../test/render-app";

const search = () => page.getByRole("combobox", { name: "Search the Indexer catalog" });
const row = (name: string) => page.getByRole("option", { name: new RegExp(`^${name}`, "u") });
const privacy = (name: string) => page.getByRole("radio", { name: new RegExp(`^${name}`, "u") });

test("the picker leads with the catalog, and offers feeds and custom definitions beside it", async () => {
  const { router } = renderApp({
    path: "/settings/indexers/new",
    handlers: addIndexerServer().handlers,
  });
  await expect
    .element(
      page.getByText("Search the Indexer catalog for the Tracker you use.", { exact: false }),
    )
    .toBeVisible();
  await expect.element(page.getByRole("link", { name: /^Torznab feed/u })).toBeVisible();
  await expect.element(page.getByRole("link", { name: /^RSS feed/u })).toBeVisible();
  await expect.element(page.getByRole("link", { name: /^Custom definition/u })).toBeVisible();
  expect(router.state.matches.at(-1)?.loaderData).toMatchObject({
    title: "Add Indexer",
    trail: [
      { title: "Settings", pathname: "/settings" },
      { title: "Indexers", pathname: "/settings/indexers" },
    ],
  });
});

test("search narrows the catalog by name, and Enter on the active row opens its form", async () => {
  const { router } = renderApp({
    path: "/settings/indexers/new",
    handlers: addIndexerServer().handlers,
  });
  await expect.element(row("Quarry")).toBeVisible();
  await expect.element(row("Quarry").getByText("Movies, TV · American English")).toBeVisible();
  await userEvent.fill(search(), "lark");
  await expect.element(row("Larkspur")).toBeVisible();
  expect(row("Quarry").query()).toBeNull();
  await userEvent.keyboard("{ArrowDown}{Enter}");
  await expect.element(page.getByRole("heading", { name: "Add Larkspur", level: 1 })).toBeVisible();
  expect(router.state.location.pathname).toBe("/settings/indexers/new/catalog/larkspur");
});

test("the filters count what the search keeps and narrow by privacy and language", async () => {
  renderApp({ path: "/settings/indexers/new", handlers: addIndexerServer().handlers });
  await expect.element(privacy("All").getByText("8", { exact: true })).toBeVisible();
  await expect.element(privacy("Private").getByText("5", { exact: true })).toBeVisible();
  await expect.element(privacy("Semi-private").getByText("1", { exact: true })).toBeVisible();
  await expect.element(privacy("Public").getByText("2", { exact: true })).toBeVisible();
  await userEvent.click(privacy("Public"));
  await expect.element(row("Cinder")).toBeVisible();
  await expect.element(row("Ferry")).toBeVisible();
  expect(row("Larkspur").query()).toBeNull();
  await userEvent.click(page.getByRole("combobox", { name: "Language" }));
  await userEvent.click(page.getByRole("option", { name: /^French/u }));
  await expect.element(row("Ferry")).toBeVisible();
  expect(row("Cinder").query()).toBeNull();
  await userEvent.fill(search(), "quarry");
  await expect.element(page.getByText("No definition matches quarry.")).toBeVisible();
  await expect.element(privacy("Public").getByText("0", { exact: true })).toBeVisible();
});

test("a definition SlipStream can't run is greyed, and its page says why under Details", async () => {
  renderApp({ path: "/settings/indexers/new", handlers: addIndexerServer().handlers });
  await expect
    .element(row("Bramble").getByText("SlipStream can't run this definition yet"))
    .toBeVisible();
  await userEvent.click(row("Bramble"));
  await expect.element(page.getByRole("heading", { name: "Add Bramble", level: 1 })).toBeVisible();
  await expect.element(page.getByText("SlipStream can't run this definition yet.")).toBeVisible();
  await userEvent.click(page.getByRole("button", { name: "Details" }));
  await expect
    .element(
      page.getByText(
        "The definition signs in with the method 'oauth', which SlipStream doesn't run.",
      ),
    )
    .toBeVisible();
  expect(page.getByRole("button", { name: "Save" }).query()).toBeNull();
});

test("a catalog definition a custom one replaces is greyed beside the custom one", async () => {
  renderApp({ path: "/settings/indexers/new", handlers: addIndexerServer().handlers });
  await userEvent.fill(search(), "marsh");
  const rows = page.getByRole("option");
  await expect.element(rows.nth(0).getByText("Replaced by your custom definition")).toBeVisible();
  await expect.element(rows.nth(1).getByText(/^Custom definition/u)).toBeVisible();
});

test("before the catalog's first download the picker says when it retries, with Refresh now", async () => {
  const runs: string[] = [];
  renderApp({
    path: "/settings/indexers/new",
    handlers: [
      http.post("*/api/v1/tasks/:taskId/runs", ({ params }) => {
        runs.push(String(params["taskId"]));
        return HttpResponse.json({ runId: 1 }, { status: 202 });
      }),
      ...addIndexerServer({ catalog: notDownloaded }).handlers,
    ],
  });
  await expect
    .element(
      page.getByText(
        "SlipStream hasn't downloaded the Indexer catalog yet. SlipStream retries at 03:00.",
      ),
    )
    .toBeVisible();
  expect(search().query()).toBeNull();
  await userEvent.click(page.getByRole("button", { name: "Refresh now" }));
  await expect.poll(() => runs).toEqual(["indexer-catalog-refresh"]);
});
