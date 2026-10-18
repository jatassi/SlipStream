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
import { type MachinesLive, machinesOf, moveMachines, stillMachines } from "./machines-live";
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

type Pools = {
  readonly links: readonly Capsules[];
  readonly boards: readonly BoardLive[];
  readonly machines: readonly MachinesLive[];
};

function stillAll({ links, boards, machines }: Pools, targets: Targets, P: Drawing["stage"]["P"]) {
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
  for (const live of machines) {
    stillMachines(live);
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
    machines: drawing.tiles
      .map((tile) => machinesOf(svg, tile, P))
      .filter((machines) => present(machines)),
  };
  const running = new Map<string, number>();
  let clock = 0;
  const runningOf = (node: string): number => running.get(node) ?? 1;
  return {
    drawing,
    frame(targets, dt) {
      clock += dt;
      for (const [node, target] of targets.running) {
        running.set(node, follow(running.get(node) ?? target, target, dt, 1.2));
      }
      for (const link of pools.links) {
        const flow = targets.flows.find(({ worker }) => worker === link.worker);
        if (flow !== undefined) {
          moveCapsules(link, flow, runningOf(link.worker), dt, P);
        }
      }
      for (const live of pools.boards) {
        const board = targets.boards.find(({ node }) => node === live.node);
        if (board !== undefined) {
          moveBoard(live, board, runningOf(live.node), dt, clock);
        }
      }
      for (const live of pools.machines) {
        const busy = targets.busy.find(({ node }) => node === live.node);
        if (busy !== undefined) {
          moveMachines(live, busy, runningOf(live.node), dt);
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
