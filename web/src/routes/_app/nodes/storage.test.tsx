import { expect, test } from "vitest";
import { page, userEvent } from "vitest/browser";

import type {
  NodesBoardNode,
  NodesNode,
  NodesStorageResource,
  RootsRoot,
} from "../../../api/gen/model";
import { getListRootsMockHandler } from "../../../api/gen/query/roots/roots.msw";
import {
  home,
  homeId,
  nodesAre,
  seedbox,
  seedboxId,
  topologyIs,
  topologyOf,
} from "../../../test/nodes";
import { renderApp } from "../../../test/render-app";
import {
  board,
  boardIs,
  homeSeat,
  outcome,
  seedboxSeat,
  writeOffServer,
} from "../../../test/roles";
import { notScanned } from "../../../test/roots";

const GB = 1024 ** 3;

const storingHome: NodesNode = {
  ...home,
  roles: [...home.roles, { role: "storage", state: "holding" }],
};

function landingWith(storage: NodesStorageResource | undefined) {
  const topology = topologyOf([storingHome, seedbox]);
  const nodes = topology.nodes.map((node) =>
    node.id === homeId && storage !== undefined ? { ...node, resources: { storage } } : node,
  );
  return renderApp({
    path: "/nodes",
    handlers: [
      nodesAre(storingHome, seedbox),
      topologyIs({ ...topology, nodes }),
      boardIs(board(homeSeat, seedboxSeat)),
    ],
  });
}

const chip = (role: string) =>
  page.elementLocator(
    document.querySelector(`[data-node="${homeId}"] [data-chip="${role}"]`) ?? document.body,
  );

test("a Storage Node with no roots yet says so in the Inspector", async () => {
  landingWith({ roots: [], volumes: [] });
  await expect.element(page.getByRole("figure", { name: "Diagram" })).toBeVisible();
  await userEvent.click(chip("storage"));
  const inspector = page.getByRole("complementary", { name: "Storage on home" });
  await expect.element(inspector.getByText(/^No library roots yet\./u)).toBeVisible();
});

const rootOf = (id: string, nodeId: string, path: string): RootsRoot => ({
  id,
  nodeId,
  path,
  contentType: "cinema",
  default: false,
  fileBytes: 6 * GB,
  state: "ready",
  works: 4,
  scan: notScanned,
});

const roots = [
  rootOf("r1", seedboxId, "/srv/media/films"),
  rootOf("r2", homeId, "/data/films"),
  rootOf("r3", seedboxId, "/srv/media/zines"),
];

const storageHeld = { units: 2, noun: "roots", bytes: 12 * GB, endlessUnits: 2 };

const drainingStorage: NodesBoardNode = {
  ...seedboxSeat,
  assigned: [],
  roles: [{ role: "storage", state: "draining", held: storageHeld }],
};

function rolesOf(seat: NodesBoardNode, writing: ReturnType<typeof writeOffServer>) {
  return renderApp({
    path: `/nodes/${seedboxId}/roles`,
    handlers: [
      nodesAre(home, {
        ...seedbox,
        standing: "in-step",
        link: { state: "up", since: "2026-10-12T10:00:00Z" },
      }),
      boardIs(board(homeSeat, seat)),
      getListRootsMockHandler({ roots }),
      writing.handler,
    ],
  });
}

test("a Node draining Storage says its roots stay", async () => {
  rolesOf(drainingStorage, writeOffServer(outcome({})));
  await expect.element(page.getByText("Finishing its work, 2 roots (12 GB) stay")).toBeVisible();
});

test("writing off Storage lists the Node's roots and says their files count as missing", async () => {
  const writing = writeOffServer(outcome({}));
  rolesOf(drainingStorage, writing);
  await userEvent.click(page.getByRole("button", { name: "Give up seedbox's work" }));
  const confirm = page.getByRole("dialog", { name: "Give up seedbox's work" });
  await expect.element(confirm.getByText("seedbox has 2 roots (12 GB) for Storage.")).toBeVisible();
  const listed = confirm.getByRole("list", { name: "Library roots on seedbox" });
  await expect.element(listed).toBeVisible();
  expect(
    listed
      .getByRole("listitem")
      .elements()
      .map((item) => item.textContent),
  ).toEqual(["/srv/media/films", "/srv/media/zines"]);
  await expect
    .element(
      confirm.getByText(
        'Its library roots stay, marked "Node given up". Their files show as missing until you edit each root to use a folder on a Node that\'s online. No files are deleted.',
      ),
    )
    .toBeVisible();
  await userEvent.click(confirm.getByRole("button", { name: "Write off" }));
  await expect.element(confirm).not.toBeInTheDocument();
  expect(writing.sent).toEqual([{ nodeId: seedboxId, body: { roles: ["storage"] } }]);
});

test("writing off torrents and roots together says what happens to each", async () => {
  const both: NodesBoardNode = {
    ...seedboxSeat,
    roles: [...seedboxSeat.roles, { role: "storage", state: "holding", held: storageHeld }],
  };
  rolesOf(both, writeOffServer(outcome({})));
  await userEvent.click(page.getByRole("button", { name: "Give up seedbox's work" }));
  const confirm = page.getByRole("dialog", { name: "Give up seedbox's work" });
  await expect
    .element(
      confirm.getByText(
        "SlipStream stops waiting for this Node. Its downloads that aren't imported yet fail, and SlipStream searches for them again. Its library roots stay, marked \"Node given up\". Their files show as missing until you edit each root to use a folder on a Node that's online. No files are deleted.",
      ),
    )
    .toBeVisible();
  await userEvent.click(
    confirm.getByRole("checkbox", { name: "3 torrents (41 GB) for Downloading" }),
  );
  await expect
    .element(
      confirm.getByText(
        'Its library roots stay, marked "Node given up". Their files show as missing until you edit each root to use a folder on a Node that\'s online. No files are deleted.',
      ),
    )
    .toBeVisible();
});
