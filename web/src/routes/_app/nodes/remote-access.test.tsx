import { HttpResponse, http } from "msw";
import { expect, test } from "vitest";
import { page, userEvent } from "vitest/browser";

import type { ReachabilityCard, ReachabilityRoute } from "../../../api/gen/model";
import { joinCodeServer } from "../../../test/join-codes";
import { homeId } from "../../../test/nodes";
import {
  cards,
  connectBack,
  connectRefused,
  dnsA,
  forward,
  plexRoute,
  remoteAccessServer,
  uiRoute,
  wanted,
} from "../../../test/remote-access";
import { renderApp } from "../../../test/render-app";

function open(card: ReachabilityCard | null, routes?: ReachabilityRoute[]) {
  const shown = wanted(card);
  const server = remoteAccessServer(routes === undefined ? shown : { ...shown, routes });
  renderApp({
    path: "/nodes/remote-access",
    handlers: [...server.handlers, ...joinCodeServer().handlers],
  });
  return server;
}

const button = (name: string) => page.getByRole("button", { name, exact: true });

test.each([
  { card: cards.fail, chip: "Needs a relay", buttons: [] },
  { card: cards.warn, chip: "Fixable", buttons: ["Turn on Open router ports automatically"] },
])("a $chip card offers only its fix as a button", async ({ card, chip, buttons }) => {
  open(card);
  const callout = page.getByRole("status").filter({ hasText: card.subject });
  await expect.element(callout.getByText(chip, { exact: true })).toBeVisible();
  expect(
    callout
      .getByRole("button")
      .elements()
      .map((element) => element.textContent),
  ).toEqual(buttons);
});

test("the map-ports fix turns on Open router ports automatically for its Node", async () => {
  const server = open(cards.warn);
  await userEvent.click(button("Turn on Open router ports automatically"));
  await expect
    .poll(() => server.sent)
    .toContainEqual({
      what: `node ${homeId}`,
      body: { values: { "reachability.mapPorts": true } },
    });
});

test("a route nothing outside tested offers to add a Node and raises no signal", async () => {
  open(cards.untested);
  await userEvent.click(button("Add a Node"));
  await expect
    .element(page.getByRole("status").filter({ hasText: "Living room Plex" }))
    .not.toBeInTheDocument();
  await expect.element(page.getByRole("dialog", { name: "Add a Node" })).toBeVisible();
});

test.each([
  { hop: "DNS", check: dnsA, leads: dnsA.meaning, meaningShown: 1 },
  { hop: "Origin", check: connectRefused, leads: forward.step, meaningShown: 0 },
  { hop: "Origin", check: connectBack, leads: connectBack.found, meaningShown: 0 },
])(
  "the $hop hop's $check.id check says what passing means only when it passes",
  async ({ hop, check, leads, meaningShown }) => {
    open(cards.warn);
    await userEvent.click(
      page
        .getByRole("list", { name: "Hops of Living room Plex" })
        .getByRole("button", { name: new RegExp(`^${hop}:`, "u") }),
    );
    await expect.element(page.getByText(leads, { exact: false })).toBeVisible();
    expect(page.getByText(check.meaning).elements()).toHaveLength(meaningShown);
  },
);

test("a failing route leaves its fix to the card and its hops, whether or not the card is its", async () => {
  const other: ReachabilityRoute = {
    ...plexRoute,
    exposure: {
      ...plexRoute.exposure,
      id: "0199f3c0-3333-7000-8000-000000000009",
      subject: "Den Plex",
    },
  };
  open(cards.warn, [plexRoute, other]);
  await expect.element(page.getByRole("region", { name: "Den Plex" })).toBeVisible();
  expect(page.getByText(forward.step, { exact: false }).elements()).toHaveLength(0);
});

test("with the Inspector open at 1440 wide, an own-proxy route's hops still run across", async () => {
  await page.viewport(1440, 900);
  const proxied: ReachabilityRoute = {
    ...plexRoute,
    route: "own-proxy",
    hops: (["player", "dns", "proxy", "tls", "origin", "media-server"] as const).map((kind, i) => ({
      kind,
      worst: i < 3 ? "passes" : i === 3 ? "fails" : "not-reached",
      checks: i === 3 ? [connectRefused] : [],
      facts: [],
    })),
  };
  open(cards.warn, [proxied]);
  const hops = page.getByRole("list", { name: "Hops of Living room Plex" }).getByRole("button");
  await userEvent.click(hops.first());
  await expect.element(page.getByRole("complementary")).toBeVisible();
  const tops = hops.elements().map((hop) => Math.round(hop.getBoundingClientRect().top));
  expect(new Set(tops).size).toBe(1);
});

const middle = (element: Element) => {
  const box = element.getBoundingClientRect();
  return Math.round(box.top + box.height / 2);
};

test.each([
  { width: 393, inspector: false },
  { width: 1440, inspector: true },
])(
  "at $width wide, inspector $inspector, a route's verdict and menu share a line, as do its family and kind",
  async ({ width, inspector }) => {
    await page.viewport(width, 900);
    const proxied: ReachabilityRoute = {
      ...plexRoute,
      route: "own-proxy",
      exposure: { ...plexRoute.exposure, public: "https://living-room-plex.proxy.example.org" },
    };
    open(cards.warn, [proxied]);
    const route = page.getByRole("region", { name: "Living room Plex" });
    await expect.element(route).toBeVisible();
    if (inspector) {
      await userEvent.click(route.getByRole("button", { name: /^Origin:/u }));
      await expect.element(page.getByRole("complementary")).toBeVisible();
    }
    const at = (locator: ReturnType<typeof route.getByText>) => middle(locator.element());
    expect(at(route.getByRole("button", { name: "Living room Plex actions" }))).toBe(
      at(route.getByText("Fixable", { exact: true })),
    );
    expect(at(route.getByText("IPv4", { exact: true }))).toBe(
      at(route.getByText("Through your own proxy", { exact: true })),
    );
  },
);

test("a route that folds after its fix closes the Inspector on its hop", async () => {
  await page.viewport(1440, 900);
  const before = wanted(cards.warn);
  let shown = before;
  const server = remoteAccessServer(before);
  renderApp({
    path: "/nodes/remote-access",
    handlers: [
      http.get("*/api/v1/remote-access", () => HttpResponse.json(shown)),
      ...server.handlers,
      ...joinCodeServer().handlers,
    ],
  });
  await userEvent.click(
    page
      .getByRole("list", { name: "Hops of Living room Plex" })
      .getByRole("button", { name: /^Origin:/u }),
  );
  await expect.element(page.getByRole("complementary")).toBeVisible();
  shown = {
    ...before,
    card: null,
    routes: [
      uiRoute,
      { ...plexRoute, verdict: "reachable", folded: true, card: "none", fix: null },
    ],
  };
  await userEvent.click(button("Turn on Open router ports automatically"));
  await expect
    .element(page.getByRole("list", { name: "Hops of Living room Plex" }))
    .not.toBeInTheDocument();
  await expect.element(page.getByRole("complementary")).not.toBeInTheDocument();
});
