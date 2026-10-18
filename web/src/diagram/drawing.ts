import { type Projection, boxCorners, fitProjection, iso } from "@anatomy/iso-kit";

import type { Landing, NodeRow } from "../routes/_app/nodes/-landing/view";
import { type TileItem, tileItems } from "./geometry";
import { AZIMUTH, type Layout, type Placement, type Route, layoutOf, planOf } from "./layout";
import { link } from "./parts/link";
import type { Item } from "./parts/part";

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
  const unit = {
    ...fitProjection(points, 1, 1, { pad: 0, azimuth: AZIMUTH }),
    origin: [0, 0] as [number, number],
    scale: 1,
  };
  const flat = points.map((point) => iso(point, unit));
  const xs = flat.map(([x]) => x);
  const ys = flat.map(([, y]) => y);
  const aspect = (Math.max(...ys) - Math.min(...ys)) / (Math.max(...xs) - Math.min(...xs));
  const height = Math.round(Math.min(Math.max(STAGE_WIDTH * aspect, 420), 900));
  return {
    P: fitProjection(points, STAGE_WIDTH, height, { pad: 24, azimuth: AZIMUTH }),
    width: STAGE_WIDTH,
    height,
  };
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

export type DrawnLink = {
  readonly route: Route;
  readonly item: Item;
  /** Whether the Worker dialed the live session; undefined while the link is down. */
  readonly dialerAtWorker: boolean | undefined;
};

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
  const dialers = new Map(landing.links.map(({ worker, dialer }) => [worker, dialer]));
  return {
    stage,
    layout,
    links: layout.links.map((route) => {
      const dialer = dialers.get(route.worker);
      return {
        route,
        item: link(stage.P, route),
        dialerAtWorker: dialer === undefined ? undefined : dialer === "worker",
      };
    }),
    tiles: layout.tiles.map((placement) => {
      const node = byId.get(placement.id);
      if (node === undefined) {
        throw new Error(`no Node ${placement.id}`);
      }
      return { node, placement, items: tileItems(tileP(stage.P, placement), node) };
    }),
  };
}
