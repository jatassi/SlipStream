import { expect, test } from "vitest";
import { page, userEvent } from "vitest/browser";

import type { IndexersIndexer } from "../../../api/gen/model";
import { larkspur, larkspurId, prowlarr, torrentLeech } from "../../../test/indexers";
import { indexersServer } from "../../../test/indexers-server";
import { iptorrents, iptorrentsId, norbits, norbitsId } from "../../../test/ports";
import { renderApp } from "../../../test/render-app";
import { asPhone } from "../../../test/viewport";

const pageOf = (id: string) => `/settings/indexers/${id}`;
const dialog = () => page.getByRole("dialog");
const codeLead = "NorBits needs the current code from your authenticator app to sign in.";
const codeField = () => page.getByRole("textbox", { name: "Two-factor code" });
const tests = (server: ReturnType<typeof indexersServer>) =>
  server.writes.filter((write) => write.method === "TEST").map((write) => write.body);
const ported = [norbits, iptorrents, torrentLeech, prowlarr, larkspur];

test("a ported site is named by its site, and its Details say where its support comes from and its spacing", async () => {
  renderApp({
    path: pageOf(iptorrentsId),
    handlers: indexersServer({ indexers: ported }).handlers,
  });
  await expect.element(page.getByRole("heading", { name: "IPTorrents", level: 1 })).toBeVisible();
  await userEvent.click(page.getByRole("button", { name: "Details" }));
  await expect
    .element(
      page.getByText(
        "SlipStream's IPTorrents support is ported from Prowlarr's code of 6 Nov 2026.",
      ),
    )
    .toBeVisible();
  await expect
    .element(page.getByText("Requests to IPTorrents are at least 6.1 seconds apart."))
    .toBeVisible();
  expect(page.getByText(/native|built-in/iu).query()).toBeNull();
});

test("the Indexers list names a ported site's Indexer by its site", async () => {
  renderApp({
    path: "/settings/indexers",
    handlers: indexersServer({ indexers: ported }).handlers,
  });
  await expect.element(page.getByText("NorBits on home")).toBeVisible();
  await expect.element(page.getByText("Catalog site on seedbox")).toBeVisible();
  expect(page.getByText(/native|built-in/iu).query()).toBeNull();
});

test("a stop for a two-factor code offers Type code, and the Test sends the code typed", async () => {
  const server = indexersServer({ indexers: ported });
  renderApp({ path: pageOf(norbitsId), handlers: server.handlers });
  await expect
    .element(page.getByText("Stopped because NorBits needs a two-factor code to sign in again."))
    .toBeVisible();
  expect(page.getByText(codeLead).query()).toBeNull();
  await userEvent.click(page.getByRole("button", { name: "Type code" }));
  await expect.element(page.getByText(codeLead)).toBeVisible();
  await expect.element(codeField()).toHaveFocus();
  expect(page.getByRole("button", { name: "Type code" }).query()).toBeNull();
  await userEvent.click(page.getByRole("button", { name: "Test" }));
  await expect.element(page.getByText("Type the code your authenticator app shows.")).toBeVisible();
  expect(tests(server)).toEqual([]);
  await userEvent.fill(codeField(), "654321");
  await userEvent.click(page.getByRole("button", { name: "Test" }));
  await expect
    .element(page.getByText("NorBits passed the Test and listed 40 items."))
    .toBeVisible();
  expect(page.getByText(codeLead).query()).toBeNull();
  expect(tests(server)).toEqual([{ code: "654321" }]);
});

test("Health's link to #code opens the page on the code step", async () => {
  renderApp({
    path: `${pageOf(norbitsId)}#code`,
    handlers: indexersServer({ indexers: ported }).handlers,
  });
  await expect.element(page.getByText(codeLead)).toBeVisible();
  await expect.element(codeField()).toHaveFocus();
});

test("a Test that meets a sign-in needing a code opens the code step at phone width too", async () => {
  await asPhone();
  const codeNeeded =
    "NorBits needs a two-factor code to sign in again. Type the current code and test again.";
  const working = { ...norbits, status: { ...norbits.status, state: "ok", cause: null } } as const;
  const server = indexersServer({ indexers: [working, prowlarr], codeNeeded });
  renderApp({ path: pageOf(norbitsId), handlers: server.handlers });
  await userEvent.click(page.getByRole("button", { name: "Test" }));
  await expect.element(page.getByRole("alert").getByText(codeNeeded)).toBeVisible();
  await expect.element(page.getByText(codeLead)).toBeVisible();
  await userEvent.fill(codeField(), "111222");
  await userEvent.click(page.getByRole("button", { name: "Test" }));
  await expect
    .element(page.getByText("NorBits passed the Test and listed 40 items."))
    .toBeVisible();
  expect(tests(server)).toEqual([{}, { code: "111222" }]);
});

test("Edit draws a ported site's address and settings, sealed ones Saved with Replace", async () => {
  const server = indexersServer({ indexers: ported });
  renderApp({ path: pageOf(norbitsId), handlers: server.handlers });
  await userEvent.click(page.getByRole("button", { name: "Edit" }));
  await expect
    .element(dialog().getByRole("combobox", { name: "Address" }))
    .toHaveTextContent("https://norbits.net/");
  await expect
    .element(
      dialog().getByText("Choose an address for NorBits, or type a new one if NorBits moves."),
    )
    .toBeVisible();
  await expect.element(dialog().getByRole("textbox", { name: "Username" })).toHaveValue("ada");
  await expect
    .element(dialog().getByRole("group", { name: "Password" }).getByText("Saved"))
    .toBeVisible();
  await expect
    .element(dialog().getByRole("group", { name: "Authenticator key" }).getByText("Saved"))
    .toBeVisible();
  await userEvent.click(dialog().getByRole("button", { name: "Save" }));
  expect(server.writes).toEqual([]);
});

test("Edit switches a ported site's mirror and replaces a sealed setting, tested before it saves", async () => {
  const failTest =
    "NorBits rejected the username or password. Update them on the Indexer and test it.";
  const server = indexersServer({ indexers: ported, failTest });
  renderApp({ path: pageOf(norbitsId), handlers: server.handlers });
  await userEvent.click(page.getByRole("button", { name: "Edit" }));
  await userEvent.click(dialog().getByRole("combobox", { name: "Address" }));
  await userEvent.click(page.getByRole("option", { name: "https://norbits.no/" }));
  await userEvent.click(dialog().getByRole("button", { name: "Replace Password" }));
  await userEvent.fill(dialog().getByLabelText("Password", { exact: true }), "hunter3");
  await userEvent.click(dialog().getByRole("button", { name: "Save" }));
  await expect.element(dialog().getByRole("alert").getByText(failTest)).toBeVisible();
  expect(server.writes).toEqual([
    {
      method: "PATCH",
      id: norbitsId,
      body: { baseAddress: "https://norbits.no/", values: { password: "hunter3" } },
    },
  ]);
});

test("Edit keeps a typed address typed, and a refused Test for a code opens the code step", async () => {
  const codeNeeded =
    "IPTorrents needs a two-factor code to sign in again. Type the current code and test again.";
  const server = indexersServer({ indexers: ported, codeNeeded });
  renderApp({ path: pageOf(iptorrentsId), handlers: server.handlers });
  await userEvent.click(page.getByRole("button", { name: "Edit" }));
  await expect
    .element(dialog().getByRole("combobox", { name: "Address" }))
    .toHaveTextContent("Another address");
  const typed = dialog().getByRole("textbox", { name: "New address" });
  await expect.element(typed).toHaveValue("https://iptorrents.example/");
  await userEvent.fill(typed, "https://iptorrents.example.net/");
  await userEvent.click(dialog().getByRole("button", { name: "Save" }));
  await expect.element(dialog().getByRole("alert").getByText(codeNeeded)).toBeVisible();
  await userEvent.fill(dialog().getByRole("textbox", { name: "Two-factor code" }), "333444");
  await userEvent.click(dialog().getByRole("button", { name: "Save" }));
  await expect.element(page.getByText("Saved IPTorrents")).toBeVisible();
  expect(server.writes.map((write) => write.body)).toEqual([
    { baseAddress: "https://iptorrents.example.net/" },
    { baseAddress: "https://iptorrents.example.net/", code: "333444" },
  ]);
});

const pinned: IndexersIndexer = {
  ...larkspur,
  definition: {
    id: "larkspur",
    version: "3f0c9a1e5b7d2c4a6e8f0b1d3c5e7a9b1d3f5a7c",
    fetchedAt: "2026-10-03T09:00:00Z",
    baseAddress: "https://larkspur.example/",
    values: { username: "orchard", sort: "added" },
  },
};

async function editCatalogSetting() {
  const server = indexersServer({ indexers: [pinned, prowlarr] });
  renderApp({ path: pageOf(larkspurId), handlers: server.handlers });
  await userEvent.click(page.getByRole("button", { name: "Edit" }));
  await expect.element(dialog().getByRole("textbox", { name: "Username" })).toHaveValue("orchard");
  await expect
    .element(dialog().getByRole("group", { name: "Password" }).getByText("Saved"))
    .toBeVisible();
  expect(dialog().getByRole("combobox", { name: "Address" }).query()).toBeNull();
  await userEvent.click(dialog().getByRole("combobox", { name: "Sort requested from site" }));
  await userEvent.click(page.getByRole("option", { name: "seeders" }));
  await userEvent.click(dialog().getByRole("button", { name: "Save" }));
  await expect.element(page.getByText("Saved Larkspur")).toBeVisible();
  expect(server.writes).toEqual([
    { method: "PATCH", id: larkspurId, body: { values: { sort: "seeders" } } },
  ]);
}

test("Edit draws a catalog Indexer's settings too, sending only the one changed", async () => {
  await editCatalogSetting();
});

test("Edit draws a catalog Indexer's settings in the phone's sheet", async () => {
  await asPhone();
  await editCatalogSetting();
});
