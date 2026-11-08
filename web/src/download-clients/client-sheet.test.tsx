import { HttpResponse } from "msw";
import { expect, test } from "vitest";
import { page, userEvent } from "vitest/browser";

import { downloadClients, homeCard } from "../test/download-clients";
import { attic, passed, refusal, withExternals } from "../test/external-clients";
import { homeId } from "../test/nodes";
import { renderApp } from "../test/render-app";

const path = "/settings/download-clients";
const sheet = (name: string) => page.getByRole("dialog", { name });
const adding = () => sheet("Add a Download Client");

async function openAdd() {
  await userEvent.click(page.getByRole("button", { name: "Add a Download Client" }));
  await expect.element(adding()).toBeVisible();
}

async function pick(label: string, option: string) {
  await userEvent.click(page.getByRole("combobox", { name: label }));
  await userEvent.click(page.getByRole("option", { name: option }));
}

const testFailed = (fields: string[], detail: string) => () =>
  refusal(422, {
    code: "download_client_test_failed",
    title: "The Download Client's Test failed",
    detail,
    fields,
  });

test("adding Transmission sends only the fields it takes, tests it on its Node and saves it onto the card", async () => {
  const clients = downloadClients();
  renderApp({ path, handlers: clients.handlers });
  await openAdd();
  await pick("Client", "Transmission");
  await expect.element(adding().getByRole("textbox", { name: "Name" })).toHaveValue("Transmission");
  await expect
    .element(
      adding().getByText(
        "The whole address, such as http://192.168.1.20:9091/transmission/rpc. With no path, SlipStream adds /transmission/rpc.",
      ),
    )
    .toBeVisible();
  expect(adding().getByText("API key").query()).toBeNull();
  await userEvent.fill(
    adding().getByRole("textbox", { name: "Address" }),
    "http://192.168.1.20:9091/transmission/rpc",
  );
  await userEvent.fill(adding().getByRole("textbox", { name: "Username" }), "jackson");
  await userEvent.fill(adding().getByLabelText("Password"), "hunter2");

  await userEvent.click(adding().getByRole("button", { name: "Test" }));
  await expect
    .element(adding().getByText("The Test passed. home reached Transmission 4.1.1."))
    .toBeVisible();
  const draft = {
    nodeId: homeId,
    kind: "transmission",
    name: "Transmission",
    address: "http://192.168.1.20:9091/transmission/rpc",
    username: "jackson",
    password: "hunter2",
    label: "slipstream",
    on: true,
    clientFolders: [],
  };
  expect(clients.writes).toEqual([{ path: "test", body: draft }]);

  await userEvent.click(adding().getByRole("button", { name: "Save" }));
  await expect.element(page.getByText("Added Transmission")).toBeVisible();
  await expect.element(adding()).not.toBeInTheDocument();
  expect(clients.writes.at(-1)).toEqual({ path: "new", body: draft });
  await expect
    .element(
      page
        .getByRole("list", { name: "External clients on home" })
        .getByRole("listitem", { name: "Transmission" }),
    )
    .toBeVisible();
});

test("each kind shows only the sign-in fields it takes", async () => {
  renderApp({ path, handlers: downloadClients().handlers });
  await openAdd();
  await expect.element(adding().getByRole("textbox", { name: "Username" })).toBeVisible();
  await expect.element(adding().getByLabelText("API key")).toBeVisible();
  await expect
    .element(adding().getByText("The whole address, such as http://192.168.1.20:8080."))
    .toBeVisible();

  await pick("Client", "Deluge");
  await expect.element(adding().getByLabelText("Password")).toBeVisible();
  expect(adding().getByRole("textbox", { name: "Username" }).query()).toBeNull();
  expect(adding().getByLabelText("API key").query()).toBeNull();
  await expect.element(adding().getByRole("textbox", { name: "Name" })).toHaveValue("Deluge");
});

test("a saved client shows its username, its secrets read Saved, and an edit sends only what changed", async () => {
  const clients = downloadClients({ cards: [withExternals(homeCard, attic)] });
  renderApp({ path, handlers: clients.handlers });
  await userEvent.click(page.getByRole("button", { name: "Edit" }));
  const editing = sheet("Edit Transmission");
  await expect.element(editing.getByRole("textbox", { name: "Username" })).toHaveValue("jackson");
  await expect.element(editing.getByText("Saved")).toBeVisible();
  await expect.element(editing.getByRole("button", { name: "Replace Password" })).toBeVisible();
  await expect.element(editing.getByRole("switch", { name: "On" })).toBeChecked();

  await userEvent.fill(editing.getByRole("textbox", { name: "Name" }), "Attic");
  await userEvent.click(editing.getByRole("button", { name: "Test" }));
  await expect
    .element(editing.getByText("The Test passed. home reached Transmission 4.1.1."))
    .toBeVisible();
  await userEvent.click(editing.getByRole("button", { name: "Save" }));
  await expect.element(page.getByText("Saved Attic")).toBeVisible();
  expect(clients.writes).toEqual([
    { path: `${attic.id}/test`, body: { name: "Attic" } },
    { path: attic.id, body: { name: "Attic" } },
  ]);
});

test("a failed Test shows its sentence and marks the fields it names, opening Details for one there", async () => {
  const clients = downloadClients({ cards: [withExternals(homeCard, attic)] });
  clients.state.answers.update = testFailed(
    ["username", "password"],
    "Transmission rejected the username or password. Check them and test again.",
  );
  renderApp({ path, handlers: clients.handlers });
  await userEvent.click(page.getByRole("button", { name: "Edit" }));
  const editing = sheet("Edit Transmission");
  await userEvent.click(editing.getByRole("button", { name: "Replace Password" }));
  await userEvent.fill(editing.getByLabelText("Password"), "wrong");
  await userEvent.click(editing.getByRole("button", { name: "Save" }));
  await expect
    .element(
      editing.getByText(
        "Transmission rejected the username or password. Check them and test again.",
      ),
    )
    .toBeVisible();
  expect(editing.getByText("Change this, then test again.").elements()).toHaveLength(2);
  await expect
    .element(editing.getByRole("textbox", { name: "Username" }))
    .toHaveAttribute("aria-invalid", "true");
  expect(clients.writes).toEqual([{ path: attic.id, body: { password: "wrong" } }]);

  clients.state.answers.test = testFailed(
    ["folder"],
    "home can't open /downloads, the folder Transmission saves to. Under Client folders, pair /downloads with the folder home sees.",
  );
  await userEvent.click(editing.getByRole("button", { name: "Test" }));
  await expect.element(editing.getByRole("textbox", { name: "Download folder" })).toBeVisible();
  await expect
    .element(editing.getByRole("textbox", { name: "Download folder" }))
    .toHaveAttribute("aria-invalid", "true");
  expect(editing.getByText("Change this, then test again.").elements()).toHaveLength(1);
});

test("a Test that can't run keeps the form and says why above Save", async () => {
  const clients = downloadClients();
  clients.state.answers.test = () =>
    refusal(502, {
      code: "download_client_unavailable",
      title: "The Test couldn't run",
      detail: "SlipStream couldn't reach home to run the Test. Test again once home is back.",
    });
  renderApp({ path, handlers: clients.handlers });
  await openAdd();
  await userEvent.fill(
    adding().getByRole("textbox", { name: "Address" }),
    "http://192.168.1.20:8080",
  );
  await userEvent.click(adding().getByRole("button", { name: "Test" }));
  await expect
    .element(
      adding().getByText(
        "SlipStream couldn't reach home to run the Test. Test again once home is back.",
      ),
    )
    .toBeVisible();
  await expect
    .element(adding().getByRole("textbox", { name: "Address" }))
    .toHaveValue("http://192.168.1.20:8080");
});

const daemons = [
  { host: "127.0.0.1", port: 58846 },
  { host: "192.168.1.30", port: 58846 },
];
const chooseDaemon =
  "Deluge's web UI lists more than one daemon. Choose the daemon SlipStream uses and test again.";

async function addDeluge() {
  await openAdd();
  await pick("Client", "Deluge");
  await userEvent.fill(
    adding().getByRole("textbox", { name: "Address" }),
    "http://192.168.1.20:8112/json",
  );
  await userEvent.fill(adding().getByLabelText("Password"), "deluge");
}

test("a Deluge whose web UI lists more than one daemon offers them and tests again with the one chosen", async () => {
  const clients = downloadClients();
  clients.state.answers.test = () =>
    refusal(422, {
      code: "download_client_choose_daemon",
      title: "Choose the daemon SlipStream uses",
      detail: chooseDaemon,
      daemons,
    });
  renderApp({ path, handlers: clients.handlers });
  await addDeluge();
  await userEvent.click(adding().getByRole("button", { name: "Test" }));
  await expect.element(adding().getByText(chooseDaemon)).toBeVisible();
  await expect.element(adding().getByRole("combobox", { name: "Daemon" })).toBeVisible();

  clients.state.answers.test = () =>
    HttpResponse.json({ ...passed, product: "Deluge", version: "2.1.1", daemons });
  await pick("Daemon", "192.168.1.30:58846");
  await userEvent.click(adding().getByRole("button", { name: "Test" }));
  await expect
    .element(adding().getByText("The Test passed. home reached Deluge 2.1.1."))
    .toBeVisible();
  expect(clients.writes.at(-1)).toEqual({
    path: "test",
    body: expect.objectContaining({
      kind: "deluge",
      daemon: { host: "192.168.1.30", port: 58846 },
    }) as unknown,
  });
  await expect
    .element(adding().getByRole("combobox", { name: "Daemon" }))
    .toHaveTextContent("192.168.1.30:58846");
});
