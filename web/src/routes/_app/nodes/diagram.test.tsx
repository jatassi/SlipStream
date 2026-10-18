import { expect, onTestFinished, test } from "vitest";
import { commands, page, userEvent } from "vitest/browser";

import type { NodesGesture } from "../../../api/gen/model";
import { fixtureOf } from "../../../diagram/fixtures";
import { takeTraffic } from "../../../live/traffic";
import { ids, vps } from "../../../test/landing";
import { home, nodesAre, seedbox, topologyIs, topologyOf } from "../../../test/nodes";
import { renderApp } from "../../../test/render-app";
import { board, boardIs, homeSeat, outcome, rolesServer, seedboxSeat } from "../../../test/roles";

const diagram = () => page.getByRole("figure", { name: "Diagram" });
const review = () => page.getByRole("dialog", { name: "Review the change" });
const chip = (node: string, role: string) => `[data-node="${node}"] [data-chip="${role}"]`;
const tile = (node: string) => `[data-node="${node}"] [data-item="tile.base"]`;

const gestures: NodesGesture[] = [
  { role: "downloading", from: ids.home, to: ids.vps, verdict: "allow", word: "Moves" },
  { role: "downloading", to: ids.vps, verdict: "allow", word: "Adds" },
  {
    role: "downloading",
    from: ids.home,
    to: ids.seedbox,
    verdict: "block",
    word: "seedbox is offline",
  },
];

const vpsSeat = { ...seedboxSeat, id: ids.vps, name: "vps", assigned: [], roles: [] };

function dragServer() {
  const server = rolesServer({ plans: [outcome({})], sets: [] });
  const handlers = [
    nodesAre(home, seedbox, vps),
    topologyIs(topologyOf([home, seedbox, vps], { rtt: 24 })),
    boardIs({ ...board(homeSeat, seedboxSeat, vpsSeat), gestures }),
    ...server.handlers,
  ];
  return { ...server, handlers };
}

const visible = (selector: string): number =>
  [...document.querySelectorAll(selector)].filter(
    (element) => Number(element.getAttribute("opacity")) > 0,
  ).length;

const textsOf = (selector: string): string[] =>
  [...document.querySelectorAll(selector)]
    .map((element) => element.textContent.trim())
    .filter((text) => text !== "");

test("dragging a Role chip onto another Node opens the Review of moving it there", async () => {
  const server = dragServer();
  renderApp({ path: "/nodes", handlers: server.handlers });
  await expect.element(diagram()).toHaveAttribute("data-takes-drags", "true");
  await commands.drag(chip(ids.home, "downloading"), tile(ids.vps));
  await expect.element(review()).toBeVisible();
  await expect.poll(() => server.planned).toHaveLength(1);
  expect(
    server.planned[0]?.set.map(({ nodeId, roles }) => [nodeId, roles.includes("downloading")]),
  ).toEqual([
    [ids.home, false],
    [ids.vps, true],
  ]);
});

test("dragging with Alt held adds the Role to the other Node and keeps it where it was", async () => {
  const server = dragServer();
  renderApp({ path: "/nodes", handlers: server.handlers });
  await expect.element(diagram()).toBeVisible();
  await userEvent.keyboard("{Alt>}");
  await commands.drag(chip(ids.home, "downloading"), tile(ids.vps));
  await userEvent.keyboard("{/Alt}");
  await expect.element(review()).toBeVisible();
  await expect.poll(() => server.planned).toHaveLength(1);
  expect(
    server.planned[0]?.set.map(({ nodeId, roles }) => [nodeId, roles.includes("downloading")]),
  ).toEqual([[ids.vps, true]]);
});

test("a drop the plan blocks opens nothing", async () => {
  const server = dragServer();
  renderApp({ path: "/nodes", handlers: server.handlers });
  await expect.element(diagram()).toBeVisible();
  await commands.drag(chip(ids.home, "downloading"), tile(ids.seedbox));
  await commands.drag(chip(ids.home, "downloading"), chip(ids.home, "searching"));
  expect(review().query()).toBeNull();
  expect(server.planned).toEqual([]);
});

test("the Roster says everything the Diagram draws, from the same Nodes", async () => {
  const { topology, board: shown } = fixtureOf(8);
  renderApp({ path: "/nodes", handlers: [nodesAre(), topologyIs(topology), boardIs(shown)] });
  await expect.element(diagram()).toBeVisible();
  const drawn = textsOf(".diagram svg text").map((text) => text.toLowerCase());
  expect(drawn.length).toBeGreaterThan(topology.nodes.length);
  await userEvent.click(
    page.getByRole("navigation", { name: "View" }).getByRole("link", { name: "Roster" }),
  );
  const roster = page.getByRole("list", { name: "Roster" });
  await expect.element(roster).toBeVisible();
  const said = roster.element().textContent.toLowerCase();
  expect(drawn.filter((text) => !said.includes(text))).toEqual([]);
});

test("under reduced motion the Diagram is one still frame: capsules spaced along the links, boards at their fill", async () => {
  await commands.emulateReducedMotion(true);
  onTestFinished(() => commands.emulateReducedMotion(false));
  const { topology, board: shown, traffic } = fixtureOf(5);
  takeTraffic(traffic[0]);
  takeTraffic(traffic[1]);
  renderApp({ path: "/nodes", handlers: [nodesAre(), topologyIs(topology), boardIs(shown)] });
  await expect.element(diagram()).toBeVisible();
  await expect.poll(() => visible(".diagram .cell")).toBeGreaterThan(0);
  expect(visible(".diagram [data-capsules] > *")).toBeGreaterThan(0);
  const frame = document.querySelector(".diagram svg")?.innerHTML;
  await new Promise<void>((resolve) => {
    setTimeout(resolve, 300);
  });
  expect(document.querySelector(".diagram svg")?.innerHTML).toBe(frame);
});

test("a click on a Node's tile opens the Inspector headed for it, and Escape closes it", async () => {
  renderApp({ path: "/nodes", handlers: dragServer().handlers });
  await expect.element(diagram()).toBeVisible();
  await userEvent.click(
    page.elementLocator(document.querySelector(tile(ids.vps)) ?? document.body),
  );
  const inspector = page.getByRole("complementary", { name: "vps" });
  await expect.element(inspector).toBeVisible();
  await userEvent.keyboard("{Escape}");
  await expect.element(inspector).not.toBeInTheDocument();
});

test("the Inspector opens below the top bar, which keeps its search and Add a Node clear of each other", async () => {
  renderApp({ path: "/nodes", handlers: dragServer().handlers });
  await expect.element(diagram()).toBeVisible();
  await userEvent.click(
    page.elementLocator(document.querySelector(tile(ids.vps)) ?? document.body),
  );
  const inspector = page.getByRole("complementary", { name: "vps" });
  await expect.element(inspector).toBeVisible();
  const bar = page.getByRole("banner").element().getBoundingClientRect();
  const search = page
    .getByRole("button", { name: "Search Works or run a command" })
    .element()
    .getBoundingClientRect();
  const add = page.getByRole("button", { name: "Add a Node" }).element().getBoundingClientRect();
  expect(inspector.element().getBoundingClientRect().top).toBeGreaterThanOrEqual(bar.bottom + 12);
  expect(search.right).toBeLessThan(add.left);
  expect(add.right).toBeLessThanOrEqual(bar.right);
});

test("a chip drags over text selected in the Diagram, which a browser would otherwise drag as text", async () => {
  const server = dragServer();
  renderApp({ path: "/nodes", handlers: server.handlers });
  await expect.element(diagram()).toBeVisible();
  document.getSelection()?.selectAllChildren(diagram().element());
  await userEvent.keyboard("{Alt>}");
  await commands.drag(chip(ids.home, "downloading"), tile(ids.vps));
  await userEvent.keyboard("{/Alt}");
  await expect.element(review()).toBeVisible();
});

test("a click on a Role chip, which a press would start dragging, opens the Inspector for that Role", async () => {
  renderApp({ path: "/nodes", handlers: dragServer().handlers });
  await expect.element(diagram()).toBeVisible();
  await userEvent.click(
    page.elementLocator(document.querySelector(chip(ids.home, "downloading")) ?? document.body),
  );
  const inspector = page.getByRole("complementary", { name: "Downloading on home" });
  await expect
    .element(inspector.getByText("home's Roles are set in its Roles section."))
    .toBeVisible();
  await userEvent.keyboard("{Escape}");
  await expect.element(inspector).not.toBeInTheDocument();
});

test("a click on a link selects the link it pressed, though the click's point misses the thin conduit", async () => {
  renderApp({ path: "/nodes", handlers: dragServer().handlers });
  await expect.element(diagram()).toBeVisible();
  const conduit = document.querySelector(`[data-link="${ids.vps}"] path`);
  const away = { bubbles: true, clientX: 0, clientY: 0, pointerId: 1, button: 0 };
  for (const type of ["pointerdown", "pointerup", "click"]) {
    conduit?.dispatchEvent(new PointerEvent(type, away));
  }
  await expect.element(page.getByRole("complementary", { name: "home and vps" })).toBeVisible();
});
