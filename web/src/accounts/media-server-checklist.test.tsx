import { HttpResponse, http } from "msw";
import { expect, test } from "vitest";
import { page, userEvent } from "vitest/browser";

import type {
  IdentityIssuedSignupLink,
  IdentityServerGrant,
  MediaserversServerView,
} from "../api/gen/model";
import {
  getListAccountsMockHandler,
  getListSignupLinksMockHandler,
} from "../api/gen/query/identity/identity.msw";
import { getListMediaServerAccessMockHandler } from "../api/gen/query/mediaservers/mediaservers.msw";
import { denJellyfin, livingRoomPlex, mediaServerHandlers } from "../test/media-servers";
import { renderApp } from "../test/render-app";

const shownPlex = { ...livingRoomPlex, membersSee: true };

const issued: IdentityIssuedSignupLink = {
  link: {
    id: "7a2e4c6b-1d3f-4a5b-9c8d-0e1f2a3b4c5d",
    email: "lin@example.test",
    issuedAt: "2026-10-14T10:00:00Z",
    expiresAt: "2026-10-21T10:00:00Z",
    mediaServers: [],
  },
  url: "https://media.example.test/signup#token=abc123",
  email: { outcome: "not-set-up" },
};

type Case = {
  readonly servers: MediaserversServerView[];
  readonly ticked: readonly string[];
  readonly off: readonly MediaserversServerView[];
  readonly pickHome?: true;
  readonly sent: IdentityServerGrant[];
};

test.each<[string, Case]>([
  [
    "shown servers are ticked, and only Plex asks how to add",
    {
      servers: [denJellyfin, shownPlex],
      ticked: [denJellyfin.name, shownPlex.name],
      off: [],
      sent: [{ server: denJellyfin.id }, { server: shownPlex.id, plexHome: false }],
    },
  ],
  [
    "a Plex server can be offered as Plex Home",
    {
      servers: [shownPlex],
      ticked: [shownPlex.name],
      off: [],
      pickHome: true,
      sent: [{ server: shownPlex.id, plexHome: true }],
    },
  ],
  [
    "a hidden server is unticked and points at its settings",
    {
      servers: [denJellyfin, livingRoomPlex],
      ticked: [denJellyfin.name],
      off: [livingRoomPlex],
      sent: [{ server: denJellyfin.id }],
    },
  ],
  ["no servers show no checklist", { servers: [], ticked: [], off: [], sent: [] }],
])("%s", async (_name, { servers, ticked, off, pickHome, sent }) => {
  const posted: unknown[] = [];
  renderApp({
    path: "/accounts?newLink=true",
    handlers: [
      getListAccountsMockHandler({ accounts: [] }),
      getListSignupLinksMockHandler({ links: [] }),
      getListMediaServerAccessMockHandler({ rows: [], plexPass: true, plexHomeUsed: false }),
      ...mediaServerHandlers(servers),
      http.post("*/api/v1/signup-links", async ({ request }) => {
        posted.push(await request.json());
        return HttpResponse.json(issued, { status: 201 });
      }),
    ],
  });
  const dialog = page.getByRole("dialog", { name: "New Signup link" });
  await expect.element(dialog.getByRole("textbox", { name: "Email" })).toBeVisible();
  const checklist = dialog.getByRole("group", { name: "Media Servers" });
  if (servers.length === 0) {
    await expect.element(checklist).not.toBeInTheDocument();
  }

  for (const server of servers) {
    const box = checklist.getByRole("checkbox", { name: server.name });
    // oxlint-disable-next-line no-await-in-loop -- each server's box in turn
    await expect.element(box).toHaveAttribute("aria-checked", String(ticked.includes(server.name)));
    // oxlint-disable-next-line no-await-in-loop -- each server's box in turn
    await (off.includes(server)
      ? expect.element(box).toBeDisabled()
      : expect.element(box).toBeEnabled());
  }
  const links = checklist.getByRole("link", { name: "Show to Members is off" });
  expect(links.elements().map((link) => link.getAttribute("href"))).toEqual(
    off.map((server) => `/settings/media-servers/${server.id}/settings#members`),
  );
  const asked = servers.filter((server) => server.kind === "plex" && ticked.includes(server.name));
  expect(checklist.getByRole("combobox", { name: "Add on Plex as" }).elements()).toHaveLength(
    asked.length,
  );

  if (pickHome) {
    await userEvent.click(checklist.getByRole("combobox", { name: "Add on Plex as" }));
    await userEvent.click(page.getByRole("option", { name: "Plex Home" }));
  }
  await userEvent.fill(dialog.getByRole("textbox", { name: "Email" }), "lin@example.test");
  await userEvent.click(dialog.getByRole("button", { name: "Make the link" }));
  await expect.element(dialog.getByRole("textbox", { name: "Signup link" })).toBeVisible();
  expect(posted).toEqual([{ email: "lin@example.test", mediaServers: sent }]);
});
