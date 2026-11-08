import { expect, test } from "vitest";
import { page, userEvent } from "vitest/browser";

import { addIndexerServer } from "../../test/add-indexer-server";
import { catalogWithPorts, portsBeforeCatalog } from "../../test/ports";
import { renderApp } from "../../test/render-app";
import { asPhone } from "../../test/viewport";

const formOf = (id: string) => `/settings/indexers/new/catalog/${id}`;
const posted = (writes: ReturnType<typeof addIndexerServer>["writes"]) =>
  writes.filter((write) => write.method === "POST").map((write) => write.body);
const search = () => page.getByRole("combobox", { name: "Search the Indexer catalog" });
const row = (name: string) => page.getByRole("option", { name: new RegExp(`^${name}`, "u") });
const privacy = (name: string) => page.getByRole("radio", { name: new RegExp(`^${name}`, "u") });
const save = () => page.getByRole("button", { name: "Save" });
const added = (name: string) =>
  expect.element(page.getByRole("heading", { name, exact: true, level: 1 })).toBeVisible();

test("ported sites sit in the catalog picker, found by the same search and filters", async () => {
  const { router } = renderApp({
    path: "/settings/indexers/new",
    handlers: addIndexerServer({ catalog: catalogWithPorts }).handlers,
  });
  await expect.element(privacy("All").getByText("14", { exact: true })).toBeVisible();
  await expect.element(privacy("Public").getByText("3", { exact: true })).toBeVisible();
  await userEvent.click(page.getByRole("combobox", { name: "Language" }));
  await userEvent.click(page.getByRole("option", { name: /^Norwegian Bokmål/u }));
  await expect.element(row("NorBits")).toBeVisible();
  expect(row("HDBits").query()).toBeNull();
  await userEvent.click(page.getByRole("combobox", { name: "Language" }));
  await userEvent.click(page.getByRole("option", { name: "Any language" }));
  await userEvent.fill(search(), "hdb");
  await expect.element(row("HDBits").getByText("Movies, TV · American English")).toBeVisible();
  expect(page.getByText(/native|built-in/iu).query()).toBeNull();
  await userEvent.keyboard("{ArrowDown}{Enter}");
  await expect.element(page.getByRole("heading", { name: "Add HDBits", level: 1 })).toBeVisible();
  expect(router.state.location.pathname).toBe(formOf("hdbits"));
});

test("before the catalog's first download, the ported sites are listed under why the rest are missing", async () => {
  renderApp({
    path: "/settings/indexers/new",
    handlers: addIndexerServer({ catalog: portsBeforeCatalog }).handlers,
  });
  await expect
    .element(
      page.getByText("SlipStream hasn't downloaded the Indexer catalog yet.", { exact: false }),
    )
    .toBeVisible();
  await expect.element(page.getByRole("button", { name: "Refresh now" })).toBeVisible();
  await expect.element(row("TorrentsCSV")).toBeVisible();
  await expect.element(row("HDBits")).toBeVisible();
});

test("a ported site's form says where fixes come from, leads with its address and sends its settings by name", async () => {
  const server = addIndexerServer();
  renderApp({ path: formOf("hdbits"), handlers: server.handlers });
  await expect
    .element(page.getByText("HDBits is a private Tracker. SlipStream searches it from home."))
    .toBeVisible();
  await expect
    .element(
      page.getByText(
        "SlipStream includes support for HDBits. Fixes to it come with SlipStream updates.",
      ),
    )
    .toBeVisible();
  const [address, username] = [
    page.getByRole("combobox", { name: "Address" }),
    page.getByRole("textbox", { name: "Username" }),
  ];
  await expect.element(address).toHaveTextContent("https://hdbits.org/");
  expect(
    address.element().compareDocumentPosition(username.element()) &
      Node.DOCUMENT_POSITION_FOLLOWING,
  ).toBeTruthy();
  await expect
    .element(page.getByText("Choose an address for HDBits, or type a new one if HDBits moves."))
    .toBeVisible();
  await expect
    .element(page.getByLabelText("Passkey", { exact: true }))
    .toHaveAttribute("type", "password");
  const codecs = page.getByRole("group", { name: "Codecs" });
  await expect.element(codecs.getByRole("checkbox", { name: "H.264" })).toBeChecked();
  await expect.element(codecs.getByRole("checkbox", { name: "HEVC" })).toBeChecked();
  await expect.element(codecs.getByRole("checkbox", { name: "VC-1" })).not.toBeChecked();

  await userEvent.fill(username, "ada");
  await userEvent.fill(page.getByLabelText("Passkey", { exact: true }), "pk-1");
  await userEvent.click(codecs.getByRole("checkbox", { name: "VC-1" }));
  await userEvent.click(codecs.getByRole("checkbox", { name: "HEVC" }));
  await userEvent.click(save());
  await added("HDBits");
  expect(posted(server.writes)).toEqual([
    expect.objectContaining({
      kind: "native",
      port: "hdbits",
      name: "HDBits",
      baseAddress: "https://hdbits.org/",
      values: { username: "ada", passkey: "pk-1", codecs: "1,3", freeleechOnly: "false" },
    }),
  ]);
});

test("a multiselect with every choice cleared is sent empty, so the site's default doesn't come back", async () => {
  const server = addIndexerServer();
  renderApp({ path: formOf("hdbits"), handlers: server.handlers });
  const codecs = page.getByRole("group", { name: "Codecs" });
  await userEvent.fill(page.getByRole("textbox", { name: "Username" }), "ada");
  await userEvent.fill(page.getByLabelText("Passkey", { exact: true }), "pk-1");
  await userEvent.click(codecs.getByRole("checkbox", { name: "H.264" }));
  await userEvent.click(codecs.getByRole("checkbox", { name: "HEVC" }));
  await userEvent.click(save());
  await added("HDBits");
  expect(posted(server.writes)).toEqual([
    expect.objectContaining({
      values: { username: "ada", passkey: "pk-1", codecs: "", freeleechOnly: "false" },
    }),
  ]);
});

test("a ported site's credentials are required, and Save names each one left empty", async () => {
  const server = addIndexerServer();
  renderApp({ path: formOf("hdbits"), handlers: server.handlers });
  const username = page.getByRole("textbox", { name: "Username" });
  const passkey = page.getByLabelText("Passkey", { exact: true });
  await expect.element(username).toHaveAttribute("aria-required", "true");
  await expect.element(passkey).toHaveAttribute("aria-required", "true");
  await userEvent.click(save());
  await expect.element(page.getByText("Username is required.")).toBeVisible();
  await expect.element(page.getByText("Passkey is required.")).toBeVisible();
  expect(posted(server.writes)).toEqual([]);
  await userEvent.fill(username, "ada");
  await userEvent.fill(passkey, "pk-1");
  await userEvent.click(save());
  await added("HDBits");
});

test("a site that moved takes a typed address, checked before the Test", async () => {
  const server = addIndexerServer();
  renderApp({ path: formOf("iptorrents"), handlers: server.handlers });
  await expect
    .element(
      page.getByText("Choose an address for IPTorrents, or type a new one if IPTorrents moves."),
    )
    .toBeVisible();
  await userEvent.click(page.getByRole("combobox", { name: "Address" }));
  await expect.element(page.getByRole("option", { name: "https://iptorrents.me/" })).toBeVisible();
  await userEvent.click(page.getByRole("option", { name: "Another address" }));
  await userEvent.fill(page.getByRole("textbox", { name: "Cookie" }), "uid=7; pass=abc");
  await userEvent.click(save());
  await expect
    .element(page.getByText("Type the address, starting with http:// or https://."))
    .toBeVisible();
  await userEvent.fill(page.getByRole("textbox", { name: "New address" }), "iptorrents.example");
  await userEvent.click(save());
  await expect
    .element(page.getByText("Type the whole address, starting with http:// or https://."))
    .toBeVisible();
  expect(posted(server.writes)).toEqual([]);
  await userEvent.fill(
    page.getByRole("textbox", { name: "New address" }),
    "https://iptorrents.example/",
  );
  await userEvent.click(save());
  await added("IPTorrents");
  expect(posted(server.writes)).toEqual([
    expect.objectContaining({
      kind: "native",
      port: "iptorrents",
      baseAddress: "https://iptorrents.example/",
      cookie: { cookie: "uid=7; pass=abc", userAgent: "" },
    }),
  ]);
});

test("a ported cookie site says which User-Agent the cookie needs", async () => {
  renderApp({ path: formOf("iptorrents"), handlers: addIndexerServer().handlers });
  await expect
    .element(
      page.getByText(
        "Use the User-Agent of the browser you copied the cookie from. IPTorrents accepts the cookie only with it.",
      ),
    )
    .toBeVisible();
});

test("a ported cookie site that takes no User-Agent asks only for the cookie, and starts at its site's limits", async () => {
  const server = addIndexerServer();
  renderApp({ path: formOf("torrentday"), handlers: server.handlers });
  await expect.element(page.getByRole("textbox", { name: "Cookie" })).toBeVisible();
  expect(page.getByRole("textbox", { name: "User-Agent" }).query()).toBeNull();
  await expect
    .element(
      page.getByText(
        "It searches from home at priority 25, with at most 100 searches a day and 20 grabs a day.",
      ),
    )
    .toBeVisible();
  await userEvent.fill(page.getByRole("textbox", { name: "Cookie" }), "uid=7");
  await userEvent.click(save());
  await added("TorrentDay");
  expect(posted(server.writes)).toEqual([
    expect.objectContaining({
      searches: { count: 100, per: "day" },
      grabs: { count: 20, per: "day" },
      cookie: { cookie: "uid=7", userAgent: "" },
    }),
  ]);
});

test("a number field shows its range and is checked against it before the Test", async () => {
  const server = addIndexerServer();
  renderApp({ path: formOf("shazbat"), handlers: server.handlers });
  const limit = page.getByRole("textbox", { name: "Show page limit" });
  await expect.element(limit).toHaveValue("2");
  await expect.element(page.getByText("From 1 to 5.")).toBeVisible();
  await userEvent.fill(page.getByLabelText("Passkey", { exact: true }), "pk-2");
  await userEvent.fill(limit, "9");
  await userEvent.click(save());
  await expect.element(page.getByText("Enter a number from 1 to 5.")).toBeVisible();
  await userEvent.fill(limit, "two");
  await userEvent.click(save());
  await expect.element(page.getByText("Enter a whole number.")).toBeVisible();
  expect(server.writes).toEqual([]);
  await userEvent.fill(limit, "4");
  await userEvent.click(save());
  await added("Shazbat");
  expect(posted(server.writes)).toEqual([
    expect.objectContaining({ values: { passkey: "pk-2", showPageLimit: "4" } }),
  ]);
});

test("a site with no list of new Releases says RSS sync can't use it and keeps RSS off", async () => {
  const server = addIndexerServer();
  renderApp({ path: formOf("torrentscsv"), handlers: server.handlers });
  await expect
    .element(page.getByText("TorrentsCSV has no list of new Releases, so RSS sync can't use it."))
    .toBeVisible();
  await userEvent.click(page.getByRole("button", { name: "Details" }));
  const rss = page.getByRole("switch", { name: "RSS" });
  await expect.element(rss).not.toBeChecked();
  await expect.element(rss).toBeDisabled();
  await userEvent.click(save());
  await added("TorrentsCSV");
  expect(posted(server.writes)).toEqual([
    expect.objectContaining({ kind: "native", port: "torrentscsv", rss: false, values: {} }),
  ]);
});

const codeNeeded =
  "NorBits needs a two-factor code to sign in. Type the current code and test again.";

async function addNorBitsWithCode() {
  const server = addIndexerServer({ codeNeeded });
  renderApp({ path: formOf("norbits"), handlers: server.handlers });
  await expect
    .element(
      page.getByText(
        "NorBits shows this key when you turn on two-factor sign-in, usually under the QR code. With it, SlipStream makes each code and signs in again automatically. Leave it empty to type a code whenever a sign-in needs one.",
      ),
    )
    .toBeVisible();
  await userEvent.fill(page.getByRole("textbox", { name: "Username" }), "ada");
  await userEvent.fill(page.getByLabelText("Password", { exact: true }), "hunter2");
  await userEvent.click(save());
  await expect.element(page.getByRole("alert").getByText(codeNeeded)).toBeVisible();
  await expect
    .element(
      page.getByText("NorBits needs the current code from your authenticator app to sign in."),
    )
    .toBeVisible();
  const code = page.getByRole("textbox", { name: "Two-factor code" });
  await expect.element(code).toHaveFocus();
  await userEvent.click(save());
  await expect.element(page.getByText("Type the code your authenticator app shows.")).toBeVisible();
  await userEvent.fill(code, "123456");
  await userEvent.click(save());
  await added("NorBits");
  const [first, second] = posted(server.writes);
  expect(first).not.toHaveProperty("code");
  expect(second).toEqual(
    expect.objectContaining({
      kind: "native",
      port: "norbits",
      values: { username: "ada", password: "hunter2" },
      code: "123456",
    }),
  );
}

test("a save whose sign-in needs a two-factor code opens the code step, and the next save sends the code", async () => {
  await addNorBitsWithCode();
});

test("the code step works the same at phone width", async () => {
  await asPhone();
  await addNorBitsWithCode();
});
