import { HttpResponse, http } from "msw";
import { expect, test } from "vitest";
import { page, userEvent } from "vitest/browser";

import type { NodesNode } from "../api/gen/model";
import { mediaServersWorld } from "../test/media-server-writes";
import { home, nodesAre, seedbox } from "../test/nodes";
import { renderApp } from "../test/render-app";

const path = "/settings/media-servers";
const dialog = () => page.getByRole("dialog");

const den: NodesNode = {
  ...seedbox,
  name: "den",
  addresses: ["192.168.1.30:8485"],
  link: { state: "up", since: "2026-11-14T09:00:00Z" },
  standing: "in-step",
  roles: [{ role: "media-server-link", state: "holding" }],
};

const address = "http://192.168.1.30:8096";

const signedIn = {
  path: "sign-in",
  body: { kind: "jellyfin", address, nodeId: den.id, user: "grace", password: "hunter2" },
};

const added = {
  path: "add",
  body: {
    kind: "jellyfin",
    name: "Jellyfin",
    address,
    apiKey: "5f1d2c3b4a5f6e7d8c9b0a1f2e3d4c5b",
    deviceId: "slipstream-0001",
    reachingNodeId: den.id,
    runsOnNodeId: den.id,
  },
};

async function openAdd(kind: string) {
  await userEvent.click(page.getByRole("button", { name: "Add a Media Server" }));
  await userEvent.click(dialog().getByRole("radio", { name: kind }));
  await userEvent.click(dialog().getByRole("button", { name: "Next" }));
  await expect.element(page.getByRole("dialog", { name: `Add ${kind}` })).toBeVisible();
}

async function signInAsAdmin() {
  await userEvent.click(dialog().getByRole("button", { name: "Sign in as an administrator" }));
  await userEvent.fill(dialog().getByRole("textbox", { name: "User" }), "grace");
  await userEvent.fill(dialog().getByLabelText("Password"), "hunter2");
  await userEvent.click(dialog().getByRole("button", { name: "Sign in", exact: true }));
}

test("adding Jellyfin signs in as its administrator, saves the key it made and moves on to its Path mappings", async () => {
  const world = mediaServersWorld([]);
  renderApp({ path, handlers: [nodesAre(home, den), ...world.handlers] });
  await expect.element(page.getByText("No Media Servers yet.", { exact: false })).toBeVisible();
  await openAdd("Jellyfin");
  await expect.element(dialog().getByRole("textbox", { name: "Name" })).toHaveValue("Jellyfin");
  await userEvent.fill(dialog().getByRole("textbox", { name: "Address" }), address);
  await expect
    .element(dialog().getByRole("combobox", { name: "Runs on" }))
    .toHaveTextContent("den");
  await expect
    .element(dialog().getByRole("combobox", { name: "Reached from" }))
    .toHaveTextContent("den");

  await signInAsAdmin();
  await expect.element(dialog().getByText("Created a key named SlipStream.")).toBeVisible();
  expect(world.writes).toEqual([signedIn]);
  await expect.element(dialog().getByLabelText("Password")).toHaveValue("");

  await userEvent.click(dialog().getByRole("button", { name: "Save" }));
  await expect.element(page.getByText("Added Jellyfin")).toBeVisible();
  expect(world.writes.at(-1)).toEqual(added);
  await expect
    .element(page.getByRole("dialog", { name: "Path mappings on Jellyfin" }))
    .toBeVisible();
  await expect
    .element(dialog().getByRole("group", { name: "Films (/srv/media/films)" }))
    .toBeVisible();
  await userEvent.click(dialog().getByRole("button", { name: "Later" }));
  await expect
    .element(
      page.getByRole("list", { name: "Media Servers" }).getByRole("link", { name: /Jellyfin/u }),
    )
    .toBeVisible();
});

test("adding Emby warns against localhost on a Worker and shows a refused name on its field", async () => {
  const world = mediaServersWorld();
  renderApp({
    path,
    handlers: [
      nodesAre(home, den),
      http.post("/api/v1/media-servers", () =>
        HttpResponse.json(
          {
            status: 409,
            code: "media_server_name_taken",
            title: "Another Media Server has this name",
          },
          { status: 409, headers: { "Content-Type": "application/problem+json" } },
        ),
      ),
      ...world.handlers,
    ],
  });
  await openAdd("Emby");
  await userEvent.fill(dialog().getByRole("textbox", { name: "Address" }), "http://localhost:8096");
  await userEvent.click(dialog().getByRole("combobox", { name: "Reached from" }));
  await userEvent.click(page.getByRole("option", { name: "den" }));
  const warning =
    "The Controller can't use localhost on den. Enter den's network address, or add a fallback address under Details.";
  await expect.element(dialog().getByText(warning)).toBeVisible();
  await userEvent.click(dialog().getByRole("button", { name: "Details" }));
  await userEvent.fill(
    dialog().getByRole("textbox", { name: "Fallback address" }),
    "http://192.168.1.30:8096",
  );
  expect(dialog().getByText(warning).query()).toBeNull();

  await userEvent.fill(dialog().getByLabelText("API key"), "e3b0c44298fc1c149afbf4c8996fb924");
  await userEvent.fill(dialog().getByRole("textbox", { name: "Name" }), "Den Jellyfin");
  await userEvent.click(dialog().getByRole("button", { name: "Save" }));
  await expect.element(dialog().getByText("Another Media Server has this name")).toBeVisible();
  await expect.element(page.getByRole("dialog", { name: "Add Emby" })).toBeVisible();
});
