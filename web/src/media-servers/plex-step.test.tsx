import { HttpResponse, http } from "msw";
import { afterEach, expect, test, vi } from "vitest";
import { page, userEvent } from "vitest/browser";

import type { MediaserversPlexServerView, NodesNode } from "../api/gen/model";
import { writePref } from "../shell/prefs";
import { livingRoomPlex, plexAccount, plexServers } from "../test/media-servers";
import { mediaServersWorld, plexPin } from "../test/media-server-writes";
import { home, nodesAre, seedbox } from "../test/nodes";
import { renderApp } from "../test/render-app";

const path = "/settings/media-servers";
const dialog = () => page.getByRole("dialog");
const polled = { timeout: 4000 };

const linked: NodesNode = {
  ...seedbox,
  link: { state: "up", since: "2026-11-14T09:00:00Z" },
  standing: "in-step",
  roles: [{ role: "media-server-link", state: "holding" }],
};

const attic: MediaserversPlexServerView = {
  machineIdentifier: "5e4d3c2b1a0f9e8d7c6b5a4f3e2d1c0b9a8f7e6d",
  name: "Attic",
  version: "1.43.4.10903",
  connections: [
    {
      uri: "https://203-0-113-9.a1b2c3.plex.direct:32400",
      local: false,
      https: true,
      reachedFrom: [
        { nodeId: home.id, ok: false },
        { nodeId: linked.id, ok: false },
      ],
    },
  ],
  suggestedReaching: { id: home.id, name: home.name },
};

function opensTab() {
  const replace = vi.fn<(url: string) => void>();
  const tab = { opener: window, location: { replace }, close: vi.fn() };
  // oxlint-disable-next-line typescript/no-unsafe-type-assertion -- a stand-in for the opened tab, with only what startPlexTab touches
  vi.spyOn(window, "open").mockReturnValue(tab as unknown as Window);
  return { replace, tab };
}

afterEach(() => {
  writePref("plexPin", null);
  vi.restoreAllMocks();
});

async function openPlex() {
  await userEvent.click(page.getByRole("button", { name: "Add a Media Server" }));
  await userEvent.click(dialog().getByRole("radio", { name: "Plex" }));
  await userEvent.click(dialog().getByRole("button", { name: "Next" }));
  await expect.element(page.getByRole("dialog", { name: "Add Plex" })).toBeVisible();
}

test("signing in to Plex opens plex.tv in a new tab, lists the account's servers once the PIN signs in, and adds the one picked", async () => {
  const world = mediaServersWorld([], null);
  const { replace, tab } = opensTab();
  renderApp({ path, handlers: [nodesAre(home, linked), ...world.handlers] });
  await openPlex();
  await userEvent.click(dialog().getByRole("button", { name: "Sign in with Plex" }));
  await expect
    .element(dialog().getByText("Waiting for you to sign in to Plex in the other tab."))
    .toBeVisible();
  expect(world.writes).toEqual([{ path: "plex-pin", body: { forward: window.location.href } }]);
  expect(replace).toHaveBeenCalledWith(plexPin.authUrl);
  expect(tab.opener).toBeNull();
  await expect
    .element(dialog().getByRole("link", { name: "Open the Plex sign-in again" }))
    .toHaveAttribute("href", plexPin.authUrl);

  await expect
    .element(dialog().getByText("Signed in to Plex as ada. The account has Plex Pass."), polled)
    .toBeVisible();
  await expect.element(dialog().getByRole("radio", { name: "Living room" })).toBeChecked();
  await expect
    .element(dialog().getByText("Plex 1.43.5.11029. Reached from seedbox."))
    .toBeVisible();
  await expect.element(dialog().getByRole("textbox", { name: "Name" })).toHaveValue("Living room");
  await expect
    .element(dialog().getByRole("combobox", { name: "Runs on" }))
    .toHaveTextContent("seedbox");
  await expect
    .element(dialog().getByRole("combobox", { name: "Reached from" }))
    .toHaveTextContent("seedbox");

  await userEvent.click(dialog().getByRole("button", { name: "Save" }));
  await expect.element(page.getByText("Added Living room")).toBeVisible();
  expect(world.writes.at(-1)).toEqual({
    path: "add",
    body: {
      kind: "plex",
      name: "Living room",
      plexServer: livingRoomPlex.identity,
      reachingNodeId: linked.id,
      runsOnNodeId: linked.id,
    },
  });
  await expect
    .element(page.getByRole("dialog", { name: "Path mappings on Living room" }))
    .toBeVisible();
});

test("a reload while the Plex tab is open opens the sheet on the waiting sign-in", async () => {
  writePref("plexPin", plexPin);
  const world = mediaServersWorld([], null);
  renderApp({ path, handlers: [nodesAre(home, linked), ...world.handlers] });
  await expect
    .element(dialog().getByText("Waiting for you to sign in to Plex in the other tab."))
    .toBeVisible();
  await expect
    .element(dialog().getByText("Signed in to Plex as ada.", { exact: false }), polled)
    .toBeVisible();
  expect(world.writes).toEqual([]);
});

test("a PIN that expires asks for the sign-in again", async () => {
  const world = mediaServersWorld([], null);
  opensTab();
  renderApp({
    path,
    handlers: [
      nodesAre(home, linked),
      http.get(`/api/v1/media-servers/plex/pin/${plexPin.pinId}`, () =>
        HttpResponse.json({ state: "expired" }),
      ),
      ...world.handlers,
    ],
  });
  await openPlex();
  await userEvent.click(dialog().getByRole("button", { name: "Sign in with Plex" }));
  await expect
    .element(
      dialog().getByText(
        "The Plex sign-in expired before it was approved. Sign in with Plex again.",
      ),
    )
    .toBeVisible();
  await expect.element(dialog().getByRole("button", { name: "Sign in with Plex" })).toBeEnabled();
});

async function openPicker() {
  const world = mediaServersWorld([], { ...plexAccount, plexPass: false });
  renderApp({
    path,
    handlers: [
      nodesAre(home, linked),
      http.get("/api/v1/media-servers/plex/servers", () =>
        HttpResponse.json({ servers: [...plexServers, attic] }),
      ),
      ...world.handlers,
    ],
  });
  await openPlex();
  return world;
}

test("picking a server no Node reached fills its own name and Nodes, and Details takes a typed address and says what Plex Pass changes", async () => {
  const world = await openPicker();
  await expect
    .element(dialog().getByText("Signed in to Plex as ada. The account doesn't have Plex Pass."))
    .toBeVisible();
  await expect
    .element(
      dialog().getByText(
        "Plex 1.43.4.10903. No Node reached it. Type an address it responds at under Details.",
      ),
    )
    .toBeVisible();
  await userEvent.click(dialog().getByRole("radio", { name: "Attic" }));
  await expect.element(dialog().getByRole("textbox", { name: "Name" })).toHaveValue("Attic");
  await expect
    .element(dialog().getByRole("combobox", { name: "Runs on" }))
    .toHaveTextContent("No Node");
  await expect
    .element(dialog().getByRole("combobox", { name: "Reached from" }))
    .toHaveTextContent("home (Controller)");

  await userEvent.click(dialog().getByRole("button", { name: "Details" }));
  const addresses = dialog().getByRole("list", { name: "Attic's addresses" });
  await expect.element(addresses.getByText(attic.connections[0]?.uri ?? "")).toBeVisible();
  await expect.element(addresses.getByText("Remote. No Node reached it.")).toBeVisible();
  await expect
    .element(dialog().getByText("Without Plex Pass, Plex ignores edition tags", { exact: false }))
    .toBeVisible();
  await userEvent.fill(
    dialog().getByRole("textbox", { name: "Address" }),
    "http://192.168.1.40:32400",
  );
  await userEvent.click(dialog().getByRole("button", { name: "Save" }));
  await expect.element(page.getByText("Added Attic")).toBeVisible();
  expect(world.writes.at(-1)).toEqual({
    path: "add",
    body: {
      kind: "plex",
      name: "Attic",
      address: "http://192.168.1.40:32400",
      plexServer: attic.machineIdentifier,
      reachingNodeId: home.id,
    },
  });
});
