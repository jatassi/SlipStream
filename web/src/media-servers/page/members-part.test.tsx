import { expect, test } from "vitest";
import { page, userEvent } from "vitest/browser";

import type { MediaserversServerView as Server } from "../../api/gen/model";
import { mediaServersWorld } from "../../test/media-server-writes";
import { denJellyfin, livingRoomPlex } from "../../test/media-servers";
import { renderApp } from "../../test/render-app";

const films = { key: "a1f0", name: "Films" };
const archive = { key: "b2e1", name: "Archive" };

test.each<{
  name: string;
  server: Server;
  change: (form: ReturnType<typeof page.getByRole>) => Promise<void>;
  sent: unknown[];
}>([
  {
    name: "refuses Show to Members with no library",
    server: denJellyfin,
    change: (form) => userEvent.click(form.getByRole("checkbox", { name: "Films" })),
    sent: [],
  },
  {
    name: "turning Show to Members on starts from the libraries behind the Path mappings",
    server: livingRoomPlex,
    change: (form) => userEvent.click(form.getByRole("switch", { name: "Show to Members" })),
    sent: [{ path: "edit", body: { membersSee: true, memberLibraries: [films, archive] } }],
  },
  {
    name: "sends a library change and the Address for Members",
    server: denJellyfin,
    change: async (form) => {
      await userEvent.click(form.getByRole("checkbox", { name: "Archive" }));
      await userEvent.fill(
        form.getByRole("textbox", { name: "Address for Members" }),
        "https://watch.example.test",
      );
    },
    sent: [
      {
        path: "edit",
        body: { memberLibraries: [films, archive], memberAddress: "https://watch.example.test" },
      },
    ],
  },
  {
    name: "sends no address on Plex, and switching off keeps the libraries",
    server: { ...livingRoomPlex, membersSee: true, memberLibraries: [films] },
    change: async (form) => {
      expect(form.getByRole("textbox", { name: "Address for Members" }).query()).toBeNull();
      await userEvent.click(form.getByRole("switch", { name: "Show to Members" }));
    },
    sent: [{ path: "edit", body: { membersSee: false } }],
  },
])("the Members part $name", async ({ server, change, sent }) => {
  const world = mediaServersWorld([server]);
  renderApp({ path: `/settings/media-servers/${server.id}/settings`, handlers: world.handlers });
  const form = page.getByRole("form", { name: `Members on ${server.name}` });
  await expect.element(form.getByRole("switch", { name: "Show to Members" })).toBeVisible();
  await change(form);
  await userEvent.click(form.getByRole("button", { name: "Save" }));
  await (sent.length === 0
    ? expect.element(form.getByRole("alert")).toBeVisible()
    : expect.poll(() => world.writes).toEqual(sent));
  expect(world.writes).toEqual(sent);
});
