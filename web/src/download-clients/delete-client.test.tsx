import { expect, test } from "vitest";
import { page, userEvent } from "vitest/browser";

import { downloadClients, homeCard } from "../test/download-clients";
import { attic, refusal, withExternals } from "../test/external-clients";
import { renderApp } from "../test/render-app";

const path = "/settings/download-clients";
const rows = () => page.getByRole("list", { name: "External clients on home" });
const confirm = () => page.getByRole("dialog", { name: "Delete Transmission?" });

async function openDelete() {
  await userEvent.click(rows().getByRole("button", { name: "Delete" }));
  await expect.element(confirm()).toBeVisible();
}

test("a client that holds no torrents SlipStream tracks deletes at once and leaves the card", async () => {
  const clients = downloadClients({ cards: [withExternals(homeCard, { ...attic, torrents: 0 })] });
  renderApp({ path, handlers: clients.handlers });
  await openDelete();
  await expect
    .element(
      confirm().getByText(
        "SlipStream stops checking Transmission. Nothing in Transmission changes.",
      ),
    )
    .toBeVisible();
  await userEvent.click(confirm().getByRole("button", { name: "Delete" }));
  await expect.element(page.getByText("Deleted Transmission")).toBeVisible();
  await expect.element(rows()).not.toBeInTheDocument();
  expect(clients.writes).toEqual([{ path: attic.id, body: null }]);
});

test("deleting a client that holds torrents says how many stay in it, stops tracking them, and a refusal shows its sentence", async () => {
  const clients = downloadClients({ cards: [withExternals(homeCard, attic)] });
  clients.state.answers.remove = () =>
    refusal(409, {
      code: "download_client_has_torrents",
      title: "This client holds torrents SlipStream tracks",
      detail: "Transmission holds 3 torrents SlipStream tracks. Stop tracking them to delete it.",
    });
  renderApp({ path, handlers: clients.handlers });
  await openDelete();
  await expect
    .element(
      confirm().getByText(
        "Transmission holds 3 torrents SlipStream tracks. They stay in Transmission, and SlipStream stops tracking them.",
      ),
    )
    .toBeVisible();
  await userEvent.click(confirm().getByRole("button", { name: "Stop tracking and delete" }));
  expect(clients.writes).toEqual([{ path: `${attic.id}?stopTracking=true`, body: null }]);
  await expect
    .element(
      confirm().getByText(
        "Transmission holds 3 torrents SlipStream tracks. Stop tracking them to delete it.",
      ),
    )
    .toBeVisible();
  await userEvent.click(confirm().getByRole("button", { name: "Cancel" }));
  await expect.element(rows().getByRole("listitem", { name: "Transmission" })).toBeVisible();
});

test("the client a Node's new Grabs go to refuses to delete with the API's sentence", async () => {
  const clients = downloadClients({
    cards: [{ ...withExternals(homeCard, { ...attic, torrents: 0 }), grabsTo: attic.id }],
  });
  clients.state.answers.remove = () =>
    refusal(409, {
      code: "download_client_in_use",
      title: "New Grabs go to this client",
      detail: "home's new Grabs go to this client. Choose another client for them first.",
    });
  renderApp({ path, handlers: clients.handlers });
  await openDelete();
  await userEvent.click(confirm().getByRole("button", { name: "Delete" }));
  await expect
    .element(
      confirm().getByText(
        "home's new Grabs go to this client. Choose another client for them first.",
      ),
    )
    .toBeVisible();
});
