import { HttpResponse, type RequestHandler, http } from "msw";
import { afterEach, expect, test, vi } from "vitest";
import { page, userEvent } from "vitest/browser";

import { readPref, writePref } from "../../shell/prefs";
import { grace, signedInAsGrace } from "../../test/handlers";
import { liveServer } from "../../test/live-server";
import {
  mediaServerAccessHandlers,
  memberPin,
  myAccess,
  myRows,
} from "../../test/media-server-access";
import { renderApp } from "../../test/render-app";
import { asPhone } from "../../test/viewport";
import { thisTab } from "./use-member-plex-pin";

const path = "/welcome/watch";
const pinUrl = `/api/v1/account/media-servers/plex/pin/${String(memberPin.pinId)}`;
const polled = { timeout: 4000 };

const { plexWaiting } = myRows;
const plexOffered = { ...plexWaiting, state: "offered" as const, actions: [] };

afterEach(() => {
  writePref("memberPlexPin", null);
  vi.restoreAllMocks();
});

function watch(rows: typeof myAccess.rows, ...handlers: RequestHandler[]) {
  return renderApp({
    path,
    handlers: [...handlers, signedInAsGrace, ...mediaServerAccessHandlers({ ...myAccess, rows })],
  });
}

function recordStart() {
  const order: string[] = [];
  const handler = http.post("/api/v1/account/media-servers/plex/pin", () => {
    order.push("pin");
    return HttpResponse.json(memberPin, { status: 201 });
  });
  const go = vi.spyOn(thisTab, "go").mockImplementation(() => {
    order.push("leave");
  });
  return { order, handler, go };
}

function opensTab(order: string[]) {
  const replace = vi.fn<(url: string) => void>();
  const tab = { opener: window, location: { replace }, close: vi.fn() };
  vi.spyOn(window, "open").mockImplementation(() => {
    order.push("tab");
    // oxlint-disable-next-line typescript/no-unsafe-type-assertion -- a stand-in for the opened tab, with only what the hook touches
    return tab as unknown as Window;
  });
  return { replace, tab };
}

test("on a wide screen the tab opens on the click, before the PIN exists, and is pointed at Plex", async () => {
  const start = recordStart();
  const { replace, tab } = opensTab(start.order);
  watch([plexOffered], start.handler);
  await userEvent.click(page.getByRole("button", { name: "Continue with Plex" }));
  await expect.poll(() => replace.mock.calls).toEqual([[memberPin.authUrl]]);
  expect(start.order).toEqual(["tab", "pin"]);
  expect(tab.opener).toBeNull();
  expect(readPref("memberPlexPin")).toEqual(memberPin);
});

test("a blocked tab says so, and the next click goes to Plex in this tab", async () => {
  const start = recordStart();
  vi.spyOn(window, "open").mockImplementation(() => {
    start.order.push("tab");
    return null;
  });
  watch([plexOffered], start.handler);
  await userEvent.click(page.getByRole("button", { name: "Continue with Plex" }));
  await expect.element(page.getByRole("status")).toBeVisible();
  expect(start.order).toEqual(["tab"]);
  await userEvent.click(page.getByRole("button", { name: "Continue with Plex" }));
  await expect.poll(() => start.go.mock.calls).toEqual([[memberPin.authUrl]]);
  expect(start.order).toEqual(["tab", "pin", "leave"]);
});

test("on a phone the page goes to Plex in this tab", async () => {
  await asPhone();
  const start = recordStart();
  const open = vi.spyOn(window, "open");
  watch([plexOffered], start.handler);
  await userEvent.click(page.getByRole("button", { name: "Continue with Plex" }));
  await expect.poll(() => start.go.mock.calls).toEqual([[memberPin.authUrl]]);
  expect(open).not.toHaveBeenCalled();
  expect(readPref("memberPlexPin")).toEqual(memberPin);
});

test("a tab that comes back from Plex picks up its sign-in, and the live event shows what changed", async () => {
  writePref("memberPlexPin", memberPin);
  const live = liveServer();
  live.become(grace);
  let signedIn = false;
  watch(
    [plexOffered],
    ...live.handlers,
    http.get(pinUrl, () => {
      if (signedIn) {
        return HttpResponse.json({ state: "signed_in" });
      }
      signedIn = true;
      return HttpResponse.json({ state: "waiting" });
    }),
    http.get("/api/v1/account/media-servers", () =>
      HttpResponse.json({ ...myAccess, rows: [signedIn ? plexWaiting : plexOffered] }),
    ),
  );
  await expect
    .element(page.getByRole("link", { name: "Open Plex's sign-in again" }))
    .toHaveAttribute("href", memberPin.authUrl);
  await expect.poll(() => readPref("memberPlexPin"), polled).toBeNull();
  live.clients[0]?.send({ event: "media-servers.my-access-changed", data: {} });
  await expect
    .element(page.getByRole("button", { name: "Use another Plex account" }))
    .toBeVisible();
});

test.each([
  ["plex_account_taken", undefined],
  ["plex_account_owner", plexOffered.server.name],
  ["plex_account_other", myAccess.plex?.username],
] as const)("a %s refusal says why and offers the sign-in again", async (refusal, fact) => {
  opensTab([]);
  watch(
    [plexOffered],
    http.get(pinUrl, () => HttpResponse.json({ state: "refused", refusal })),
  );
  await userEvent.click(page.getByRole("button", { name: "Continue with Plex" }));
  const line = page.getByRole("status");
  await expect.element(line, polled).toBeVisible();
  if (fact !== undefined) {
    expect(line.element().textContent).toContain(fact);
  }
  await expect.element(page.getByRole("button", { name: "Continue with Plex" })).toBeEnabled();
  expect(readPref("memberPlexPin")).toBeNull();
});
