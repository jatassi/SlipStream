import { type Point3, type Projection, boxCorners, fitProjection, iso } from "@anatomy/iso-kit";

import type { RoleName } from "../nodes/roles";
import type { Landing, NodeRow } from "../routes/_app/nodes/-landing/view";
import { KEYS } from "./keys.gen";
import { AZIMUTH, type Layout, type Placement, type Route, TOP, layoutOf, planOf } from "./layout";
import { board } from "./parts/board";
import { cabinet } from "./parts/cabinet";
import { link } from "./parts/link";
import { type Item, movedP, placed } from "./parts/part";
import { press } from "./parts/press";
import { base, chip, lamp, plaque, CHIP_SLOTS } from "./parts/tile";
import { tower } from "./parts/tower";

/** The Roles with a machine, and where it stands on the tile's top. */
const MACHINES: Partial<Record<RoleName, { readonly at: Point3; readonly draw: (P: Projection) => Item[] }>> = {
  downloading: { at: [0, 0, TOP], draw: board },
  storage: { at: [0, 0, TOP], draw: (P) => [cabinet(P)] },
  converting: { at: [-152, -62, TOP], draw: (P) => [press(P)] },
  relaying: { at: [60, -30, TOP], draw: (P) => [tower(P)] },
};

export const machineRoles = Object.keys(MACHINES) as readonly RoleName[];

export type TileItem = Item & { readonly role?: RoleName };

/** Every item a tile draws for a Node, in tile coordinates, unsorted. */
export function itemsOf(P: Projection, node: NodeRow): TileItem[] {
  const chips = node.roles
    .slice(0, CHIP_SLOTS)
    .map((role, slot) => ({ ...chip(P, slot, role.label, role.note), role: role.role }));
  const machines = node.roles.flatMap(({ role }) => {
    const machine = MACHINES[role];
    if (machine === undefined) {
      return [];
    }
    return machine.draw(movedP(P, machine.at)).map((it) => ({ ...placed(it, machine.at), role }));
  });
  return [base(P), plaque(P, node.name, node.plaque), lamp(P), ...chips, ...machines];
}

/** A tile's items in painter's order. */
export const tileItems = (P: Projection, node: NodeRow): TileItem[] =>
  itemsOf(P, node).toSorted((a, b) => keyOf(a.name) - keyOf(b.name));

export function keyOf(name: string): number {
  const key = KEYS[name];
  if (key === undefined) {
    throw new Error(`no depth key for ${name}`);
  }
  return key;
}

/** The tallest a tile's machines stand above the floor, for framing. */
const HEADROOM = 175;

export const STAGE_WIDTH = 1200;

export type Stage = {
  readonly P: Projection;
  readonly width: number;
  readonly height: number;
};

/** The projection that fits every tile, with room above for the machines. */
export function stageOf(layout: Layout): Stage {
  const points = layout.tiles.flatMap((tile) => boxCorners(planOf(tile), 0, HEADROOM * tile.scale));
  const unit = fitProjection(points, 1, 1, { pad: 0, azimuth: AZIMUTH });
  const flat = points.map((point) => iso(point, { ...unit, origin: [0, 0], scale: 1 }));
  const xs = flat.map(([x]) => x);
  const ys = flat.map(([, y]) => y);
  const aspect = (Math.max(...ys) - Math.min(...ys)) / (Math.max(...xs) - Math.min(...xs));
  const height = Math.round(Math.min(Math.max(STAGE_WIDTH * aspect, 420), 900));
  return { P: fitProjection(points, STAGE_WIDTH, height, { pad: 24, azimuth: AZIMUTH }), width: STAGE_WIDTH, height };
}

/** A tile's own projection: its local coordinates drawn at its place and scale. */
export const tileP = (P: Projection, { at: [x, y], scale }: Placement): Projection => ({
  ...P,
  scale: P.scale * scale,
  origin: iso([x, y, 0], P),
});

export type DrawnTile = {
  readonly node: NodeRow;
  readonly placement: Placement;
  readonly items: readonly TileItem[];
};

export type DrawnLink = { readonly route: Route; readonly item: Item };

export type Drawing = {
  readonly stage: Stage;
  readonly layout: Layout;
  /** Floor first: the links, then the tiles back to front. */
  readonly links: readonly DrawnLink[];
  readonly tiles: readonly DrawnTile[];
};

/** Everything the Diagram draws for a landing, in painter's order. */
export function drawingOf(landing: Landing): Drawing {
  const [controller, ...workers] = landing.nodes;
  if (controller === undefined) {
    throw new Error("a landing always has its Controller");
  }
  const layout = layoutOf(
    controller.id,
    workers.map(({ id }) => id),
  );
  const stage = stageOf(layout);
  const byId = new Map(landing.nodes.map((node) => [node.id, node]));
  return {
    stage,
    layout,
    links: layout.links.map((route) => ({ route, item: link(stage.P, route) })),
    tiles: layout.tiles.map((placement) => {
      const node = byId.get(placement.id);
      if (node === undefined) {
        throw new Error(`no Node ${placement.id}`);
      }
      return { node, placement, items: tileItems(tileP(stage.P, placement), node) };
    }),
  };
}
