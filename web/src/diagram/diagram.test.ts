import { describe, expect, test } from "vitest";

import { takeTraffic, trafficPair } from "../live/traffic";
import { landingOf } from "../routes/_app/nodes/-landing/view";
import { drawingOf } from "./drawing";
import { fixtureClock, fixtureOf } from "./fixtures";
import { MOST_NODES, layoutOf, planOf } from "./layout";
import {
  capsuleRate,
  litCells,
  pointOn,
  shareOf,
  stillCapsules,
  targetsOf,
  trackOf,
} from "./model";

const landingFrom = (count: number) => {
  const { topology, board } = fixtureOf(count);
  return landingOf(topology, board, new Set(), fixtureClock);
};

const overlaps = (a: ReturnType<typeof planOf>, b: ReturnType<typeof planOf>) =>
  a.x < b.x + b.w && b.x < a.x + a.w && a.y < b.y + b.d && b.y < a.y + a.d;

describe("layout", () => {
  test.each([1, 2, 5, 8])(
    "%i Nodes: the Controller at the centre, a smaller tile for each Worker, one conduit to each and no two tiles touching",
    (count) => {
      const landing = landingFrom(count);
      const { layout, tiles, links } = drawingOf(landing);
      const [controller, ...workers] = landing.nodes;
      expect(layout.tiles.find(({ id }) => id === controller?.id)).toMatchObject({
        at: [0, 0],
        scale: 1,
      });
      expect(tiles.map(({ node }) => node.id).toSorted()).toEqual(
        landing.nodes.map(({ id }) => id).toSorted(),
      );
      expect(tiles.filter(({ placement }) => placement.scale < 1)).toHaveLength(workers.length);
      expect(links.map(({ route }) => route.worker)).toEqual(workers.map(({ id }) => id));
      const plans = layout.tiles.map((tile) => planOf(tile));
      for (const [index, plan] of plans.entries()) {
        expect(plans.slice(index + 1).filter((other) => overlaps(plan, other))).toEqual([]);
      }
    },
  );

  test("tiles are painted back to front", () => {
    const depth = drawingOf(landingFrom(8)).tiles.map(({ placement: { at } }) => at[0] + at[1]);
    expect(depth).toEqual(depth.toSorted((a, b) => a - b));
  });

  test("past the most Nodes the Diagram draws, the rest are left to the Roster", () => {
    const workers = Array.from({ length: MOST_NODES + 4 }, (_, index) => `w${String(index)}`);
    const layout = layoutOf("c", workers);
    expect(layout.tiles).toHaveLength(MOST_NODES);
    expect(layout.links).toHaveLength(MOST_NODES - 1);
    expect(layout.unplaced).toBe(5);
  });
});

describe("machines", () => {
  test("a tile draws a machine only for the Roles its Node holds, and a chip for every Role", () => {
    for (const { node, items } of drawingOf(landingFrom(12)).tiles) {
      const held = new Set(node.roles.map(({ role }) => role));
      const drawn = new Set(items.flatMap(({ role }) => (role === undefined ? [] : [role])));
      expect([...drawn].filter((role) => !held.has(role))).toEqual([]);
      expect(items.filter(({ name }) => name.startsWith("chip."))).toHaveLength(node.roles.length);
      expect(items.some(({ name }) => name.startsWith("downloading."))).toBe(
        held.has("downloading"),
      );
      expect(items.some(({ name }) => name === "storage")).toBe(held.has("storage"));
    }
  });

  test("an away Node and a draining Role keep their tile and chip, and say so", () => {
    const landing = landingFrom(5);
    const attic = landing.nodes.find(({ name }) => name === "attic");
    const garage = landing.nodes.find(({ name }) => name === "garage");
    expect(attic?.state).toBe("offline");
    expect(attic?.plaque).toMatch(/^Offline since /u);
    expect(garage?.roles.find(({ role }) => role === "downloading")?.state).toBe("draining");
    const tiles = drawingOf(landing).tiles;
    expect(tiles.find(({ node }) => node.id === attic?.id)?.items.length).toBeGreaterThan(1);
    expect(
      drawingOf(landing).links.find(({ route }) => route.worker === attic?.id)?.dialerAtWorker,
    ).toBeUndefined();
  });
});

describe("motion", () => {
  test("a link's capsule rate follows its throughput, capped by what its round trip can carry", () => {
    expect(capsuleRate(0, 20)).toBe(0);
    expect(capsuleRate(1_000_000, 0)).toBe(0);
    expect(capsuleRate(1_000_000, 20)).toBeCloseTo(2);
    expect(capsuleRate(1e9, 20)).toBeCloseTo(6.25);
    expect(capsuleRate(1e9, 200)).toBeLessThan(capsuleRate(1e9, 20));
  });

  test("between two samples the figures move from the earlier to the later over the gap between them", () => {
    const [before, latest] = fixtureOf(2).traffic;
    expect(shareOf([before, latest], 0)).toBe(0);
    expect(shareOf([before, latest], 500)).toBe(0.5);
    expect(shareOf([before, latest], 5000)).toBe(1);
    expect(shareOf([undefined, latest], 0)).toBe(1);
  });

  test("an online link carries capsules, a down one none, and a Node away winds its machines down", () => {
    const { topology, board, traffic } = fixtureOf(5);
    const landing = landingOf(topology, board, new Set(), fixtureClock);
    const targets = targetsOf(landing, traffic, Number.POSITIVE_INFINITY);
    const attic = landing.nodes.find(({ name }) => name === "attic")?.id ?? "";
    for (const flow of targets.flows) {
      expect(flow.rate > 0).toBe(flow.worker !== attic);
    }
    expect(targets.running.get(attic)).toBe(0);
    const home = targets.boards.find(({ node }) => node === landing.nodes[0]?.id);
    expect(home?.panels.length).toBeGreaterThan(0);
    expect(home?.pieces).toBeGreaterThan(0);
  });
});

describe("still frame", () => {
  test("the still frame spaces capsules a capsule's travel apart inside the ends, and lights whole and part cells", () => {
    const placed = stillCapsules(400, { worker: "w", rate: 2, speed: 100, towardWorker: 0.5 }, 28);
    expect(placed).toEqual([12, 62, 112, 162, 212, 262, 312, 362]);
    expect(stillCapsules(400, { worker: "w", rate: 0, speed: 100, towardWorker: 0.5 }, 28)).toEqual(
      [],
    );
    expect(litCells(0.5, 35)).toEqual({ whole: 17, part: 0.5 });
    expect(litCells(2, 35)).toEqual({ whole: 35, part: 0 });
  });

  test("a point along a track is measured along its length and clamped to its ends", () => {
    const track = trackOf([
      [0, 0, 0],
      [10, 0, 0],
      [10, 10, 0],
    ]);
    expect(track.length).toBe(20);
    expect(pointOn(track, 15)).toEqual([10, 5, 0]);
    expect(pointOn(track, 40)).toEqual([10, 10, 0]);
    expect(pointOn(track, -1)).toEqual([0, 0, 0]);
  });

  test("the traffic ring keeps the latest sample and the one before it", () => {
    const [first, second] = fixtureOf(2).traffic;
    takeTraffic(first);
    takeTraffic(second);
    takeTraffic(first);
    expect(trafficPair()).toEqual([second, first]);
  });
});
