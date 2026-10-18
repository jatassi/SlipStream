import { type Point3, type Projection, iso, pathOf, translateAlong } from "@anatomy/iso-kit";
import { follow } from "@anatomy/react/live";

import { type DrawnTile, tileP } from "./drawing";
import { TOP } from "./layout";
import { type Board, type Track, litCells, pointOn, smooth, trackOf } from "./model";
import { BEAM_Z, CHUTE_IN, PANELS, SHUTTLE_HOME, cellCentre, railOf } from "./parts/board";
import { fixed, hide, make, write } from "./svg";

// A piece board's motion: pieces down the chute from the antenna, each
// dropped by the shuttle into the next cell of its torrent's panel, and the
// cells lit to each torrent's fill.

const PIECES = 10;
const PIECE_SPEED = 130;
const GRAVITY = 520;

type Piece = { s: number; alive: boolean; shown: boolean; readonly el: SVGPathElement };
type Drop = {
  readonly el: SVGRectElement;
  readonly key: string;
  age: number;
  readonly z0: number;
  readonly z1: number;
};

export type BoardLive = {
  readonly node: string;
  readonly P: Projection;
  readonly cells: readonly (readonly SVGRectElement[])[];
  readonly drops: readonly (Element | null)[];
  readonly sliders: readonly (Element | null)[];
  readonly shuttle: Element | null;
  readonly rx: Element | null;
  readonly beacon: Element | null;
  readonly chute: Track;
  readonly pieces: readonly Piece[];
  falling: Drop[];
  readonly flashes: Map<string, number>;
  readonly written: WeakMap<Element, Map<string, string>>;
  arriving: number;
  next: number;
  shuttleAt: number;
  rate: number;
  fills: readonly number[];
};

function cubeOf(P: Projection): string {
  const flat: Projection = { ...P, origin: [0, 0] };
  const corners: readonly (readonly [number, number])[] = [
    [-1.1, -1.1],
    [1.1, -1.1],
    [1.1, 1.1],
    [-1.1, 1.1],
  ];
  const top = corners.map(([x, y]) => iso([x, y, 2.2], flat));
  const bottom = corners.map(([x, y]) => iso([x, y, 0], flat));
  return pathOf(
    [top[0], top[1], bottom[1], bottom[2], bottom[3], top[3]].filter(
      (point) => point !== undefined,
    ),
    true,
  );
}

const each = <T>(select: (panel: number) => string, read: (selector: string) => T): T[] =>
  PANELS.map((_, panel) => read(select(panel)));

export function boardOf(
  svg: SVGSVGElement,
  { node, placement }: DrawnTile,
  stage: Projection,
): BoardLive | undefined {
  const tile = svg.querySelector(`[data-node="${CSS.escape(node.id)}"]`);
  const layer = tile?.querySelector("[data-pieces]") ?? null;
  if (tile === null || layer === null) {
    return undefined;
  }
  const P = tileP(stage, placement);
  const cube = cubeOf(P);
  return {
    node: node.id,
    P,
    cells: each(
      (panel) => `[data-cell^="${String(panel)}-"]`,
      (selector) => [...tile.querySelectorAll<SVGRectElement>(selector)],
    ),
    drops: each(
      (panel) => `[data-drops="${String(panel)}"]`,
      (selector) => tile.querySelector(selector),
    ),
    sliders: each(
      (panel) => `[data-progress="${String(panel)}"]`,
      (selector) => tile.querySelector(selector),
    ),
    shuttle: tile.querySelector("[data-shuttle]"),
    rx: tile.querySelector('[data-led="rx"]'),
    beacon: tile.querySelector('[data-led="beacon"]'),
    chute: trackOf(CHUTE_IN.map(([x, y, z]): Point3 => [x, y, z + TOP])),
    pieces: Array.from({ length: PIECES }, () => ({
      s: 0,
      alive: false,
      shown: false,
      el: make("path", { class: "piece", d: cube, opacity: "0" }, layer),
    })),
    falling: [],
    flashes: new Map(),
    written: new WeakMap(),
    arriving: 0,
    next: 0,
    shuttleAt: SHUTTLE_HOME,
    rate: 0,
    fills: [],
  };
}

/** The cell the next piece of a panel lands in. */
function nextCell(live: BoardLive, panel: number): number {
  const shape = PANELS[panel] ?? PANELS[0];
  const cells = shape.cols * shape.rows;
  return Math.min(cells - 1, Math.floor((live.fills[panel] ?? 0) * cells));
}

function paintCells(live: BoardLive, now: number) {
  live.cells.forEach((cells, panel) => {
    const fill = live.fills[panel];
    const { whole, part } = litCells(fill ?? 0, cells.length);
    cells.forEach((cell, index) => {
      const lit = index < whole ? 0.55 : index === whole ? part * 0.35 : 0;
      const flashed = live.flashes.get(`${String(panel)}-${String(index)}`);
      const flash = flashed === undefined ? 0 : Math.exp(-(now - flashed) / 0.35);
      write(
        live.written,
        cell,
        "opacity",
        fill === undefined ? "0" : (lit + flash * 0.6).toFixed(2),
      );
    });
    const rail = railOf(PANELS[panel] ?? PANELS[0]);
    write(
      live.written,
      live.sliders[panel] ?? null,
      "transform",
      translateAlong(live.P, [(fill ?? 0) * rail, 0, 0]),
    );
  });
}

function land(live: BoardLive) {
  if (live.fills.length === 0) {
    return;
  }
  const panel = live.next % live.fills.length;
  live.next++;
  const layer = live.drops[panel];
  const shape = PANELS[panel];
  if (layer === null || layer === undefined || shape === undefined) {
    return;
  }
  const cell = nextCell(live, panel);
  const centre = cellCentre(shape, cell);
  const el = make(
    "rect",
    { class: "drop", x: fixed(centre.u - 1.6), y: "0", width: "3.2", height: "3.2", rx: "0.6" },
    layer,
  );
  live.falling.push({
    el,
    key: `${String(panel)}-${String(cell)}`,
    age: 0,
    z0: BEAM_Z - 4,
    z1: centre.z + 3.7,
  });
}

function movePieces(live: BoardLive, dt: number) {
  for (const piece of live.pieces) {
    if (piece.alive) {
      piece.s += PIECE_SPEED * dt;
    }
    if (!piece.alive || piece.s >= live.chute.length) {
      if (piece.alive) {
        land(live);
      }
      piece.alive = false;
      hide(piece, piece.el);
      continue;
    }
    const [x, y] = iso(pointOn(live.chute, piece.s), live.P);
    piece.el.setAttribute("transform", `translate(${fixed(x)} ${fixed(y)})`);
    piece.el.setAttribute(
      "opacity",
      (smooth(0, 10, piece.s) * smooth(0, 6, live.chute.length - piece.s)).toFixed(3),
    );
    piece.shown = true;
  }
}

function moveDrops(live: BoardLive, dt: number, now: number) {
  live.falling = live.falling.filter((drop) => {
    drop.age += dt;
    const z = drop.z0 - 0.5 * GRAVITY * drop.age * drop.age;
    if (z > drop.z1) {
      drop.el.setAttribute("y", fixed(-z - 1.6));
      return true;
    }
    live.flashes.set(drop.key, now);
    drop.el.remove();
    return false;
  });
}

/** One frame of a board: ease to its rate, take in pieces, move them, drop them and light the cells. */
export function moveBoard(live: BoardLive, board: Board, running: number, dt: number, now: number) {
  live.fills = board.panels;
  live.rate = follow(live.rate, board.pieces * running, dt, 0.6);
  for (live.arriving += live.rate * dt; live.arriving >= 1; live.arriving -= 1) {
    const free = live.pieces.find((piece) => !piece.alive);
    if (free !== undefined) {
      free.alive = true;
      free.s = 0;
    }
  }
  movePieces(live, dt);
  moveDrops(live, dt, now);
  const panel = live.fills.length > 0 ? live.next % live.fills.length : -1;
  const shape = PANELS[panel];
  const target =
    shape === undefined || live.rate < 0.05
      ? SHUTTLE_HOME
      : cellCentre(shape, nextCell(live, panel)).u;
  live.shuttleAt = follow(live.shuttleAt, target, dt, 0.35);
  write(
    live.written,
    live.shuttle,
    "transform",
    translateAlong(live.P, [live.shuttleAt - SHUTTLE_HOME, 0, 0]),
  );
  const busy = Math.min(live.rate / 3, 1);
  write(
    live.written,
    live.rx,
    "opacity",
    (0.25 + 0.75 * busy * (0.6 + 0.4 * Math.sin(now * 17))).toFixed(2),
  );
  write(
    live.written,
    live.beacon,
    "opacity",
    (0.4 + 0.6 * running * (0.5 + 0.5 * Math.sin(now * 2))).toFixed(2),
  );
  paintCells(live, now);
}

/** The still frame of a board: its fill, nothing in flight, the shuttle home. */
export function stillBoard(live: BoardLive, board: Board | undefined) {
  live.fills = board?.panels ?? [];
  for (const piece of live.pieces) {
    piece.alive = false;
    hide(piece, piece.el);
  }
  for (const drop of live.falling) {
    drop.el.remove();
  }
  live.falling = [];
  live.flashes.clear();
  write(live.written, live.shuttle, "transform", "");
  paintCells(live, 0);
}

export function disposeBoard(live: BoardLive) {
  for (const piece of live.pieces) {
    piece.el.remove();
  }
  for (const drop of live.falling) {
    drop.el.remove();
  }
}
