import { expect, test } from "vitest";
import { page, userEvent } from "vitest/browser";

import type { IndexersIndexer } from "../../../api/gen/model";
import {
  larkspur,
  larkspurId,
  prowlarr,
  prowlarrId,
  torrentLeech,
  torrentLeechId,
} from "../../../test/indexers";
import { indexersServer } from "../../../test/indexers-server";
import { homeId, seedboxId } from "../../../test/nodes";
import { renderApp } from "../../../test/render-app";

const pageOf = (id: string) => `/settings/indexers/${id}`;
const dialog = () => page.getByRole("dialog");

test("the page leads with the status sentence and its fix, with the machinery under Details", async () => {
  const { router } = renderApp({ path: pageOf(larkspurId), handlers: indexersServer().handlers });
  await expect.element(page.getByRole("heading", { name: "Larkspur", level: 1 })).toBeVisible();
  await expect
    .element(page.getByText("Stopped because Larkspur rejected the username or password."))
    .toBeVisible();
  await expect
    .element(page.getByText("Update the username or password with Edit. Saving tests them."))
    .toBeVisible();
  await expect.element(page.getByText("Found 12 Releases")).toBeVisible();
  await expect.element(page.getByText(".torrent fetch")).toBeVisible();
  expect(page.getByText("Not signed in on any Node").query()).toBeNull();
  await userEvent.click(page.getByRole("button", { name: "Details" }));
  await expect.element(page.getByText("Not signed in on any Node")).toBeVisible();
  expect(router.state.matches.at(-1)?.loaderData).toMatchObject({
    title: "Larkspur",
    trail: [
      { title: "Settings", pathname: "/settings" },
      { title: "Indexers", pathname: "/settings/indexers" },
    ],
  });
});

test("the page shows the Trackers it reaches and how much of each limit is used", async () => {
  renderApp({ path: pageOf(torrentLeechId), handlers: indexersServer().handlers });
  await expect.element(page.getByText("TorrentLeech is working.")).toBeVisible();
  await expect.element(page.getByText("4 of 10 per hour")).toBeVisible();
  await expect.element(page.getByRole("meter", { name: "Searches used" })).toBeVisible();
  await expect.element(page.getByRole("switch", { name: "Allow failover" })).toBeVisible();
  await expect.element(page.getByText("seedbox", { exact: true }).first()).toBeVisible();
});

test("Test reports what the Indexer listed, or the sentence of why it failed", async () => {
  renderApp({ path: pageOf(prowlarrId), handlers: indexersServer().handlers });
  await userEvent.click(page.getByRole("button", { name: "Test" }));
  await expect
    .element(page.getByText("Prowlarr passed the Test and listed 40 items."))
    .toBeVisible();
});

test("a failed Test shows the cause's sentence", async () => {
  const failTest =
    "Prowlarr refused the API key with 'Invalid API Key'. Check the account, then test again.";
  renderApp({ path: pageOf(prowlarrId), handlers: indexersServer({ failTest }).handlers });
  await userEvent.click(page.getByRole("button", { name: "Test" }));
  await expect.element(page.getByRole("alert").getByText(failTest)).toBeVisible();
});

test("Edit shows secrets as Saved, sends only what changed, and a new key is tested before it saves", async () => {
  const failTest =
    "Prowlarr refused the API key with 'Invalid API Key'. Check the account, then test again.";
  const server = indexersServer({ failTest });
  renderApp({ path: pageOf(prowlarrId), handlers: server.handlers });
  await userEvent.click(page.getByRole("button", { name: "Edit" }));
  await expect
    .element(dialog().getByRole("group", { name: "Feed address" }).getByText("prowlarr.lan:9696"))
    .toBeVisible();
  await expect
    .element(dialog().getByRole("group", { name: "API key" }).getByText("Saved"))
    .toBeVisible();

  await userEvent.click(dialog().getByRole("button", { name: "Replace API key" }));
  await userEvent.fill(dialog().getByLabelText("API key", { exact: true }), "wrong-key");
  await userEvent.click(dialog().getByRole("button", { name: "Save" }));
  await expect.element(dialog().getByRole("alert").getByText(failTest)).toBeVisible();
  expect(server.writes.at(-1)).toEqual({
    method: "PATCH",
    id: prowlarrId,
    body: { apiKey: "wrong-key" },
  });
});

test("Edit saves the priority and a limit without testing", async () => {
  const server = indexersServer();
  renderApp({ path: pageOf(prowlarrId), handlers: server.handlers });
  await userEvent.click(page.getByRole("button", { name: "Edit" }));
  await userEvent.fill(dialog().getByRole("textbox", { name: "Priority" }), "10");
  await userEvent.fill(dialog().getByRole("textbox", { name: "Searches", exact: true }), "20");
  await userEvent.click(dialog().getByRole("radio", { name: "per day" }).first());
  await userEvent.click(dialog().getByRole("button", { name: "Save" }));
  await expect.element(page.getByText("0 of 20 per day")).toBeVisible();
  expect(server.writes).toEqual([
    {
      method: "PATCH",
      id: prowlarrId,
      body: { priority: 10, searches: { count: 20, per: "day" } },
    },
  ]);
});

test("moving an Indexer that signs in warns of the new address, then saves its home", async () => {
  const server = indexersServer();
  renderApp({ path: pageOf(torrentLeechId), handlers: server.handlers });
  await userEvent.click(page.getByRole("button", { name: "Move" }));
  await userEvent.click(dialog().getByRole("combobox", { name: "Search from" }));
  await userEvent.click(page.getByRole("option", { name: "home" }));
  await expect
    .element(
      dialog().getByText(
        "TorrentLeech will see a sign-in from home's address. It may ask for a captcha or email you about it.",
      ),
    )
    .toBeVisible();
  await userEvent.click(dialog().getByRole("button", { name: "Move" }));
  await expect.element(page.getByText("TorrentLeech searches from home now")).toBeVisible();
  expect(server.writes).toEqual([
    { method: "PATCH", id: torrentLeechId, body: { homeNodeId: homeId } },
  ]);
});

test("delete keeps the Trackers and returns to the list", async () => {
  const server = indexersServer();
  const { router } = renderApp({ path: pageOf(prowlarrId), handlers: server.handlers });
  await userEvent.click(page.getByRole("button", { name: "Delete" }));
  await expect
    .element(
      dialog().getByText(
        "Searches stop asking it. Its Trackers stay, and so does the history of the Grabs it made.",
      ),
    )
    .toBeVisible();
  await userEvent.click(dialog().getByRole("button", { name: "Delete" }));
  await expect.poll(() => router.state.location.pathname).toBe("/settings/indexers");
  await expect.element(page.getByText("Deleted Prowlarr")).toBeVisible();
  expect(server.writes).toEqual([{ method: "DELETE", id: prowlarrId }]);
  expect(page.getByRole("link", { name: "Prowlarr" }).query()).toBeNull();
});

test("an Indexer waiting on a lapsed home leads with Allow failover", async () => {
  const waiting: IndexersIndexer = {
    ...torrentLeech,
    sentence: "TorrentLeech is waiting for seedbox, which has been offline since 08:00.",
    fix: "Allow failover to search from another Node.",
    waits: {
      kind: "waiting",
      releases: 0,
      until: null,
      since: "2026-10-26T08:00:00Z",
      nodeId: seedboxId,
      lapsed: true,
      cause: null,
      text: null,
    },
  };
  const server = indexersServer({ indexers: [waiting, prowlarr, larkspur] });
  renderApp({ path: pageOf(torrentLeechId), handlers: server.handlers });
  await expect.element(page.getByText("Allow failover to search from another Node.")).toBeVisible();
  const lead = page.getByRole("switch", { name: "Allow failover" }).first();
  expect(document.querySelector("#allow-failover [role=switch]")).not.toBeNull();
  await userEvent.click(lead);
  await expect
    .poll(() => server.writes.at(-1))
    .toEqual({
      method: "PATCH",
      id: torrentLeechId,
      body: { allowFailover: true },
    });
});
