import { follow } from "@anatomy/react/live";

import { type BoardLive, boardOf, disposeBoard, moveBoard, stillBoard } from "./boards";
import {
  type Capsules,
  capsulesOf,
  disposeCapsules,
  moveCapsules,
  stillCapsulesOn,
} from "./capsules";
import type { Drawing } from "./drawing";
import type { Targets } from "./model";

// The Diagram's motion, written straight to the SVG the art drew. It never
// re-renders React: a frame only writes attributes on the elements the art
// drew and on the pools added here.

export type Engine = {
  readonly drawing: Drawing;
  /** One frame: ease toward the targets over dt seconds and move everything on. */
  readonly frame: (targets: Targets, dt: number) => void;
  /** The still frame for reduced motion: capsules evenly spaced, boards at their fill, nothing in flight. */
  readonly still: (targets: Targets) => void;
  /** Takes the pools out of the SVG. */
  readonly dispose: () => void;
};

const present = <T>(value: T | undefined): value is T => value !== undefined;

type Pools = { readonly links: readonly Capsules[]; readonly boards: readonly BoardLive[] };

function stillAll({ links, boards }: Pools, targets: Targets, P: Drawing["stage"]["P"]) {
  for (const link of links) {
    stillCapsulesOn(
      link,
      targets.flows.find(({ worker }) => worker === link.worker),
      P,
    );
  }
  for (const live of boards) {
    stillBoard(
      live,
      targets.boards.find(({ node }) => node === live.node),
    );
  }
}

function disposeAll({ links, boards }: Pools) {
  for (const link of links) {
    disposeCapsules(link);
  }
  for (const live of boards) {
    disposeBoard(live);
  }
}

export function engineOf(svg: SVGSVGElement, drawing: Drawing): Engine {
  const { P } = drawing.stage;
  const pools: Pools = {
    links: drawing.links
      .map((link) => capsulesOf(svg, link, P.scale))
      .filter((link) => present(link)),
    boards: drawing.tiles.map((tile) => boardOf(svg, tile, P)).filter((board) => present(board)),
  };
  const running = new Map<string, number>();
  let clock = 0;
  const runningOf = (targets: Targets, node: string, dt: number): number => {
    const target = targets.running.get(node) ?? 1;
    const next = follow(running.get(node) ?? target, target, dt, 1.2);
    running.set(node, next);
    return next;
  };
  return {
    drawing,
    frame(targets, dt) {
      clock += dt;
      for (const link of pools.links) {
        const flow = targets.flows.find(({ worker }) => worker === link.worker);
        if (flow !== undefined) {
          moveCapsules(link, flow, runningOf(targets, link.worker, dt), dt, P);
        }
      }
      for (const live of pools.boards) {
        const board = targets.boards.find(({ node }) => node === live.node);
        if (board !== undefined) {
          moveBoard(live, board, runningOf(targets, live.node, dt), dt, clock);
        }
      }
    },
    still(targets) {
      stillAll(pools, targets, P);
    },
    dispose() {
      disposeAll(pools);
    },
  };
}
