import type { Plan, Plane } from "@anatomy/iso-kit";
import { describe, expect, test } from "vitest";

import { type TrafficPair, takeTraffic, trafficPair } from "../live/traffic";
import { landingOf } from "../routes/_app/nodes/-landing/view";
import { drawingOf } from "./drawing";
import type { TileItem } from "./geometry";
import { fixtureClock, fixtureOf } from "./fixtures";
import { MOST_NODES, TOP, layoutOf, planOf } from "./layout";
import {
  DRAWER_HOLD,
  QUIET,
  SWEEP_HOLD,
  TRICKLE,
  capsuleRate,
  litCells,
  pointOn,
  shareOf,
  stillCapsules,
  targetsOf,
  trackOf,
} from "./model";
import { chipPlan } from "./parts/tile";

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

const machinesOn = (items: readonly TileItem[]): TileItem[] =>
  items.filter(({ role, name }) => role !== undefined && !name.startsWith("chip."));

/** The floor a machine's solids cover, as the box around them. */
function footprintOf(items: readonly TileItem[]): Plan {
  const points = items
    .flatMap(({ shapes }) => shapes)
    .flatMap((shape): readonly Plane[] => {
      if (shape.kind === "slab") {
        const { x, y, w, d } = shape.plan;
        return [
          [x, y],
          [x + w, y + d],
        ];
      }
      if (shape.kind === "extruded") {
        return shape.ring;
      }
      const { o, a, u, v } = shape.frame;
      const along = (s: number, p: number, q: number): Plane => [
        o[0] + a[0] * s + u[0] * p + v[0] * q,
        o[1] + a[1] * s + u[1] * p + v[1] * q,
      ];
      const polygon: readonly Plane[] =
        shape.kind === "prism"
          ? shape.polygon
          : [
              [-shape.r, -shape.r],
              [shape.r, shape.r],
              [-shape.r, shape.r],
              [shape.r, -shape.r],
            ];
      return [shape.s0, shape.s1].flatMap((s) => polygon.map(([p, q]) => along(s, p, q)));
    });
  const xs = points.map(([x]) => x);
  const ys = points.map(([, y]) => y);
  const x = Math.min(...xs);
  const y = Math.min(...ys);
  return { x, y, w: Math.max(...xs) - x, d: Math.max(...ys) - y, r: 0 };
}

const footprintsOf = (machines: readonly TileItem[]) =>
  [...new Set(machines.map(({ role }) => role))].map((role) => ({
    name: String(role),
    plan: footprintOf(machines.filter((machine) => machine.role === role)),
  }));

describe("machines", () => {
  test("a tile draws a machine for each Role its Node holds and none other, and a chip for every Role", () => {
    for (const { node, items } of drawingOf(landingFrom(12)).tiles) {
      const held = node.roles.map(({ role }) => role).toSorted((a, b) => a.localeCompare(b));
      const drawn = new Set(machinesOn(items).map(({ role }) => role));
      expect([...drawn].toSorted((a, b) => (a ?? "").localeCompare(b ?? ""))).toEqual(held);
      expect(items.filter(({ name }) => name.startsWith("chip."))).toHaveLength(node.roles.length);
    }
  });

  test("a lone Controller holds all ten Roles, and their machines stand apart and behind the chips", () => {
    const [tile] = drawingOf(landingFrom(1)).tiles;
    const machines = machinesOn(tile?.items ?? []);
    expect(new Set(machines.map(({ role }) => role)).size).toBe(10);
    const plans = footprintsOf(machines);
    const chips = Math.min(...[0, 5].map((slot) => chipPlan(slot).y));
    for (const [index, { name, plan }] of plans.entries()) {
      expect(
        plans.slice(index + 1).filter((other) => overlaps(plan, other.plan)),
        name,
      ).toEqual([]);
      expect(plan.y + plan.d, name).toBeLessThan(chips);
    }
  });

  test("every machine is cabled into its tile: one cable per Role, dropping onto a grommet set on the tile's top", () => {
    for (const { node, items } of drawingOf(landingFrom(12)).tiles) {
      for (const { role } of node.roles) {
        const own = items.filter((item) => item.role === role);
        const cables = own
          .flatMap(({ pipes }) => pipes)
          .filter(({ name }) => name === `${role}.lead.cable`);
        expect(cables, role).toHaveLength(1);
        const grommet = own
          .flatMap(({ shapes }) => shapes)
          .find(({ name }) => name === `${role}.lead.grommet`);
        expect(grommet?.kind, role).toBe("disc");
        if (grommet?.kind !== "disc") {
          continue;
        }
        const end = cables[0]?.points.at(-1);
        expect(grommet.frame.o[2] + grommet.s0, role).toBe(TOP);
        expect(end?.[2], role).toBeCloseTo(grommet.frame.o[2] + grommet.s1);
        expect(
          Math.hypot((end?.[0] ?? 0) - grommet.frame.o[0], (end?.[1] ?? 0) - grommet.frame.o[1]),
          role,
        ).toBeCloseTo(0);
      }
    }
  });

  test("a machine stands in the same place on every tile that holds its Role", () => {
    const tiles = drawingOf(landingFrom(12)).tiles;
    const places = new Map<string, string>();
    for (const machine of tiles.flatMap(({ items }) => machinesOn(items))) {
      const place = JSON.stringify(machine.at);
      expect(places.get(machine.name) ?? place, machine.name).toBe(place);
      places.set(machine.name, place);
    }
  });

  test("an away Node and a draining Role keep their tile and chip, and say so", () => {
    const landing = landingFrom(5);
    const attic = landing.nodes.find(({ name }) => name === "attic");
    const garage = landing.nodes.find(({ name }) => name === "garage");
    expect(attic?.state).toBe("offline");
    expect(attic?.plaque).toMatch(/^Last seen /u);
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
    expect(capsuleRate(68, 0)).toBe(TRICKLE);
    expect(capsuleRate(1_000_000, 0)).toBeCloseTo(2);
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
    const targets = targetsOf(landing, traffic, Number.POSITIVE_INFINITY, QUIET);
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

function busyOn(count: number, search: number, fetch: number) {
  const { topology, board, traffic } = fixtureOf(count);
  const landing = landingOf(topology, board, new Set(), fixtureClock);
  const { busy } = targetsOf(landing, traffic, 0, { search, fetch });
  return landing.nodes.flatMap(({ id, name, roles }) => {
    const held = new Set(roles.map(({ role }) => role));
    const seen = busy.find(({ node }) => node === id);
    return seen === undefined ? [] : [{ name, held, ...seen }];
  });
}

describe("work", () => {
  test("while a search reports, every Node holding Searching sweeps its dish, and no other", () => {
    const busy = busyOn(12, 1, Number.POSITIVE_INFINITY);
    expect(busy.length).toBeGreaterThan(1);
    for (const { name, held, sweep, drawer } of busy) {
      expect(sweep, name).toBe(held.has("searching") ? 1 : 0);
      expect(drawer, name).toBe(0);
    }
  });

  test("a dish stops sweeping once no search has reported for a while", () => {
    expect(busyOn(1, SWEEP_HOLD - 0.1, Number.POSITIVE_INFINITY)[0]?.sweep).toBe(1);
    expect(busyOn(1, SWEEP_HOLD, Number.POSITIVE_INFINITY)[0]?.sweep).toBe(0);
  });

  test("a fetch slides out the drawer of every Node holding Metadata, which goes home after", () => {
    for (const { name, held, drawer } of busyOn(12, Number.POSITIVE_INFINITY, 0.2)) {
      expect(drawer, name).toBe(held.has("metadata") ? 1 : 0);
    }
    expect(busyOn(1, Number.POSITIVE_INFINITY, DRAWER_HOLD)[0]?.drawer).toBe(0);
  });

  test("with no work reported, every machine is at rest", () => {
    for (const { sweep, drawer } of busyOn(
      12,
      Number.POSITIVE_INFINITY,
      Number.POSITIVE_INFINITY,
    )) {
      expect([sweep, drawer]).toEqual([0, 0]);
    }
  });
});

describe("still frame", () => {
  test("a board takes pieces in at least at a trickle while its Node downloads anything, and none while it is idle", () => {
    const { topology, board, traffic } = fixtureOf(1);
    const landing = landingOf(topology, board, new Set(), fixtureClock);
    const downloading = (downBps: number): TrafficPair => {
      const [, latest] = traffic;
      return [
        undefined,
        {
          ...latest,
          nodes: latest.nodes.map((node) => Object.assign(structuredClone(node), { downBps })),
        },
      ];
    };
    expect(targetsOf(landing, downloading(1000), 0, QUIET).boards[0]?.pieces).toBe(TRICKLE);
    expect(targetsOf(landing, downloading(0), 0, QUIET).boards[0]?.pieces).toBe(0);
  });

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
