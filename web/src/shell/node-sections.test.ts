import { expect, test } from "vitest";

import type { NodesRoleViewState } from "../api/gen/model";
import { home, seedbox } from "../test/nodes";
import { sectionsFor } from "./node-sections";

const downloading = (state: NodesRoleViewState) => ({
  ...home,
  roles: [
    { role: "searching", state: "holding" } as const,
    { role: "downloading", state } as const,
  ],
});

const ids = (node: Parameters<typeof sectionsFor>[0], nodes = [node]) =>
  sectionsFor(node, nodes).map(({ id }) => id);

test("a Node without Downloading has no Torrent client section", () => {
  expect(ids({ ...home, roles: [{ role: "searching", state: "holding" }] })).toEqual([
    "overview",
    "roles",
    "web",
    "logs",
    "log-settings",
  ]);
});

test.each(["holding", "away", "lapsed", "covering", "draining", "writing-off"] as const)(
  "a Node %s Downloading has the Torrent client section after Web and API",
  (state) => {
    expect(ids(downloading(state))).toEqual([
      "overview",
      "roles",
      "web",
      "torrent-client",
      "logs",
      "log-settings",
    ]);
  },
);

test("with more than one Node, each Node has Transfers after Torrent client", () => {
  expect(ids(home, [home, seedbox])).toEqual([
    "overview",
    "roles",
    "web",
    "torrent-client",
    "transfers",
    "logs",
    "log-settings",
  ]);
  expect(ids(seedbox, [home, seedbox])).toContain("transfers");
  expect(ids(home)).not.toContain("transfers");
});
