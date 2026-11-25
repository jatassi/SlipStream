import { HttpResponse, http } from "msw";
import { afterEach, expect, test } from "vitest";
import { page, userEvent } from "vitest/browser";

import { writePref } from "../shell/prefs";
import { plexAccount } from "../test/media-servers";
import { mediaServersWorld, plexPin } from "../test/media-server-writes";
import { home, nodesAre, seedbox } from "../test/nodes";
import { renderApp } from "../test/render-app";

const pinPath = `/api/v1/media-servers/plex/pin/${String(plexPin.pinId)}`;
const dialog = () => page.getByRole("dialog");

afterEach(() => {
  writePref("plexPin", null);
});

test.each([
  { press: "Use this Plex account", replaced: [plexPin.pinId], step: "servers" },
  { press: "Back", replaced: [], step: "kinds" },
])(
  "a sign-in as a different Plex account asks first, and $press replaces it $replaced.length times",
  async ({ press, replaced, step }) => {
    writePref("plexPin", plexPin);
    const calls: number[] = [];
    const world = mediaServersWorld([], { ...plexAccount, refused: true });
    renderApp({
      path: "/settings/media-servers",
      handlers: [
        nodesAre(home, seedbox),
        http.get(pinPath, () => HttpResponse.json({ state: "other_account" })),
        http.post(`${pinPath}/replace`, () => {
          calls.push(plexPin.pinId);
          return HttpResponse.json(plexAccount);
        }),
        ...world.handlers,
      ],
    });
    await userEvent.click(dialog().getByRole("button", { name: press }));
    await expect.poll(() => calls).toEqual(replaced);
    await expect
      .element(
        step === "servers"
          ? dialog().getByRole("radiogroup", { name: "Plex server" })
          : dialog().getByRole("radio", { name: "Plex" }),
      )
      .toBeVisible();
  },
);
