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

const TB = 1024 ** 4;
const GB = 1024 ** 3;

const storingHome: NodesNode = {
  ...home,
  roles: [...home.roles, { role: "storage", state: "holding" }],
};

const figures: NodesStorageResource = {
  roots: [
    { id: "r1", contentType: "cinema", path: "/mnt/media/films", state: "ready", volumeId: "v1" },
    { id: "r2", contentType: "zine", path: "/mnt/media/zines", state: "ready", volumeId: "v1" },
    { id: "r3", contentType: "cinema", path: "/mnt/fast/films", state: "ready", volumeId: "v2" },
  ],
  volumes: [
    { ...volumeOf("v1", "/mnt/media"), total: 4 * TB, free: 1 * TB },
    { ...volumeOf("v2", "/mnt/fast"), total: 2 * TB, free: 0.2 * TB },
  ],
};

function volumeOf(id: string, mount: string) {
  return { id, name: mount, mount, kind: "local", sampledAt: "2026-10-12T10:00:00Z" } as const;
}

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

const usedOf = (index: number): string | undefined =>
  document.querySelector<SVGGElement>(
    `[data-node="${homeId}"] [data-role="storage"] [data-drawer="${String(index)}"]`,
  )?.dataset["used"];

const chip = (role: string) =>
  page.elementLocator(
    document.querySelector(`[data-node="${homeId}"] [data-chip="${role}"]`) ?? document.body,
  );

test("the cabinet's drawers read its volumes' used shares, fullest at the top, and the rest stand empty", async () => {
  landingWith(figures);
  await expect.element(page.getByRole("figure", { name: "Diagram" })).toBeVisible();
  await expect
    .poll(() => [3, 2, 1, 0].map((index) => usedOf(index)))
    .toEqual(["0.90", "0.75", undefined, undefined]);
});

test("the Storage Inspector counts the roots, their volumes and the free space, and links to Library roots", async () => {
  landingWith(figures);
  await expect.element(page.getByRole("figure", { name: "Diagram" })).toBeVisible();
  await userEvent.click(chip("storage"));
  const inspector = page.getByRole("complementary", { name: "Storage on home" });
  await expect.element(inspector.getByText(/^3 roots on 2 volumes, 1\.2 TB free\./u)).toBeVisible();
  await expect
    .element(inspector.getByRole("link", { name: "Library roots" }))
    .toHaveAttribute("href", "/settings/library-roots");
  expect(inspector.getByText("No live figures yet").query()).toBeNull();
});

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
});

const roots = [
  rootOf("r1", seedboxId, "/srv/media/films"),
  rootOf("r2", homeId, "/data/films"),
  rootOf("r3", seedboxId, "/srv/media/zines"),
];

const storageHeld = { units: 2, noun: "roots", bytes: 12 * GB, endless: true };

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
  await expect.element(page.getByText("Draining, 2 roots (12 GB) stay")).toBeVisible();
});

test("writing off Storage lists the Node's roots and says their files count as missing", async () => {
  const writing = writeOffServer(outcome({}));
  rolesOf(drainingStorage, writing);
  await userEvent.click(page.getByRole("button", { name: "Write off what seedbox holds" }));
  const confirm = page.getByRole("dialog", { name: "Write off what seedbox holds" });
  await expect
    .element(confirm.getByText("seedbox holds 2 roots (12 GB) for Storage."))
    .toBeVisible();
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
        "Its library roots stay, marked Node written off, and their files count as missing until you re-home the roots onto a live Node. No files are deleted.",
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
  await userEvent.click(page.getByRole("button", { name: "Write off what seedbox holds" }));
  const confirm = page.getByRole("dialog", { name: "Write off what seedbox holds" });
  await expect
    .element(
      confirm.getByText(
        "SlipStream stops waiting for it. What isn't imported yet goes back to wanted, so SlipStream searches for it again. Its library roots stay, marked Node written off, and their files count as missing until you re-home the roots onto a live Node. No files are deleted.",
      ),
    )
    .toBeVisible();
  await userEvent.click(
    confirm.getByRole("checkbox", { name: "3 torrents (41 GB) for Downloading" }),
  );
  await expect
    .element(
      confirm.getByText(
        "Its library roots stay, marked Node written off, and their files count as missing until you re-home the roots onto a live Node. No files are deleted.",
      ),
    )
    .toBeVisible();
});
