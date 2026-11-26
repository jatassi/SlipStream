import type { RequestHandler } from "msw";
import { expect, test } from "vitest";
import { page } from "vitest/browser";

import type { MediaserversMyAccessRow as MyRow } from "../../api/gen/model";
import { signedInAsGrace } from "../../test/handlers";
import { mediaServerAccessHandlers, myAccess, myRows } from "../../test/media-server-access";
import { renderApp } from "../../test/render-app";

const path = "/welcome/watch";

const { plexWaiting, jellyfinOffered } = myRows;

function watch(rows: MyRow[], ...handlers: RequestHandler[]) {
  return renderApp({
    path,
    handlers: [...handlers, signedInAsGrace, ...mediaServerAccessHandlers({ ...myAccess, rows })],
  });
}

const buttonsIn = (name: string) =>
  page
    .getByRole("region", { name })
    .getByRole("button")
    .all()
    .map((button) => button.element().textContent);

test.each<[string, MyRow, string[]]>([
  ["Plex waiting", plexWaiting, ["Use another Plex account"]],
  ["Plex waiting, nothing to start", { ...plexWaiting, actions: [] }, []],
  [
    "Plex shared",
    { ...plexWaiting, state: "shared", actions: ["check"] },
    ["Sign in with Plex again", "Check again"],
  ],
  [
    "Plex shared, nothing to start",
    { ...plexWaiting, state: "shared", actions: [] },
    ["Sign in with Plex again"],
  ],
  [
    "Jellyfin offered, linked by the Admin only",
    {
      ...jellyfinOffered,
      server: { ...jellyfinOffered.server, access: "users_linked" },
      actions: [],
    },
    [],
  ],
  ["Jellyfin making", { ...jellyfinOffered, state: "making", actions: [] }, []],
])(
  "%s draws the buttons its actions name, and Skip for now while it is pending",
  async (_, row, buttons) => {
    watch([row]);
    await expect.element(page.getByRole("region", { name: row.server.name })).toBeVisible();
    expect(buttonsIn(row.server.name)).toEqual(buttons);
    const [shown, hidden] = row.pending
      ? ["Skip for now", "Continue"]
      : ["Continue", "Skip for now"];
    await expect.element(page.getByRole("button", { name: shown, exact: true })).toBeVisible();
    await expect
      .element(page.getByRole("button", { name: hidden, exact: true }))
      .not.toBeInTheDocument();
  },
);
