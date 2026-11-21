import { expect, test } from "vitest";

import {
  filmsStep,
  homeRow,
  nodeRow,
  reconcilingPage,
  restore,
  seedboxRow,
} from "../test/reconciling";
import { finishWords, nextOf, nodeState } from "./words";

const clock = { now: new Date("2026-11-21T09:00:00Z"), zone: "UTC" };

test("the next step leads with pairing, then the folders, then the scans", () => {
  const page = reconcilingPage({
    restore: { ...restore, rehome: true },
    nodes: [homeRow, seedboxRow, nodeRow({ id: "n3", name: "nas", link: "in-step" })],
    roots: [filmsStep, { ...filmsStep, id: "r2", label: "Music library root on home" }],
  });
  expect(nextOf(page, clock)).toEqual({
    kind: "steps",
    sentence:
      "Pair seedbox again, find the old Controller's folders, wait for nas to report its torrents and wait for 2 library roots to finish scanning.",
  });
});

test("when only Nodes offline past the grace are left, it recommends continuing without them", () => {
  const laptop = nodeRow({
    id: "n4",
    name: "laptop",
    link: "lapsed",
    linkSince: "2026-11-10T08:30:00Z",
  });
  const page = reconcilingPage({
    restore: { ...restore, newMachine: false },
    nodes: [homeRow, laptop],
    roots: [{ ...filmsStep, scanned: true }],
  });
  expect(nextOf(page, clock)).toEqual({
    kind: "offline",
    sentence:
      "SlipStream is waiting only for laptop, which has been offline since Nov 10, 08:30. Continue without it so searches start again.",
    nodes: [laptop],
  });
});

test("an offline Node beside other waits is one step among them", () => {
  const laptop = nodeRow({ id: "n4", name: "laptop", link: "lapsed" });
  const page = reconcilingPage({
    restore: { ...restore, newMachine: false },
    nodes: [homeRow, laptop],
  });
  expect(nextOf(page, clock).sentence).toBe(
    "Continue without laptop and wait for 1 library root to finish scanning.",
  );
});

test("Finish now names every wait it ends and the Nodes that keep their torrents", () => {
  expect(finishWords(reconcilingPage())).toBe(
    "Finish Reconciling now? SlipStream stops waiting for seedbox and the Films library root on home. seedbox keeps its torrents until it reports them.",
  );
});

test("a Node only reconnecting after a restore on a new machine still needs pairing", () => {
  const seedbox = { ...seedboxRow, link: "reconnecting" } as const;
  expect(nodeState(seedbox, true)).toBe("pair");
  expect(nodeState(seedbox, false)).toBe("offline");
});
