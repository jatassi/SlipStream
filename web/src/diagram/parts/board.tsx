import {
  type Plan,
  type Plane,
  type Point3,
  type Projection,
  iso,
  lineOnTop,
  onTop,
} from "@anatomy/iso-kit";
import { fillet, frameOf } from "@anatomy/lathe";

import {
  type Drawn,
  type Item,
  box,
  dots,
  group,
  halo,
  item,
  line,
  mark,
  owned,
  prism,
  screws,
} from "./part";
import { faceY, roundRect } from "./face";
import { RX, antenna } from "./antenna";
import { tube } from "./tube";

// Downloading's machine: an antenna takes pieces in, two chutes carry them to
// a hopper on a loom, and the loom's shuttle drops each into the board of the
// torrent it belongs to. Coordinates are on the tile's top, at z = 0.

export const LOOM = { x: -122, y: -106, w: 120, d: 20, r: 2 };
export const BOARD_Y = -90;
export const BEAM_Z = 86;
export const HOPPER = { x: -108, y: -101, w: 14, d: 10, r: 1 };
export const CELL = { w: 9, h: 9.4, gap: 0.8, z0: 13 };
export const PANELS = [
  { x0: -112, cols: 5, rows: 7 },
  { x0: -60, cols: 5, rows: 7 },
] as const;

export type Panel = (typeof PANELS)[number];

const panelTop = (panel: Panel): number => CELL.z0 + panel.rows * (CELL.h + CELL.gap) - CELL.gap;
const panelEnd = (panel: Panel): number => panel.x0 + panel.cols * (CELL.w + CELL.gap) - CELL.gap;

/** Where a cell of a panel sits on the board's face: its centre along x, and its centre height. */
export function cellCentre(
  panel: Panel,
  index: number,
): { readonly u: number; readonly z: number } {
  const row = Math.floor(index / panel.cols);
  const column = index % panel.cols;
  return {
    u: panel.x0 + column * (CELL.w + CELL.gap) + CELL.w / 2,
    z: panelTop(panel) - (row + 1) * CELL.h - row * CELL.gap + CELL.h / 2,
  };
}

/** How far a panel's progress slider runs along its rail. */
export const railOf = (panel: Panel): number => panelEnd(panel) - panel.x0;

const HOPPER_TOP = BEAM_Z + 15;
const INTO: Plane = [HOPPER.x + HOPPER.w / 2, HOPPER.y + HOPPER.d / 2];

/** Where the shuttle rests, under the hopper; the live loop moves it along x from here. */
export const SHUTTLE_HOME = INTO[0];

/** The chute from the antenna's receiver over to the hopper, dropping into its open top. */
export const CHUTE_IN: readonly Point3[] = fillet(
  [
    [RX.x + RX.w, RX.y + 7, RX.z + 8],
    [INTO[0], RX.y + 7, RX.z + 8],
    [INTO[0], INTO[1], RX.z + 8],
    [INTO[0], INTO[1], HOPPER_TOP],
  ],
  5,
  8,
);
export const CHUTE_R = 2.1;

function panelFace(panel: Panel, index: number): Drawn {
  const x1 = panelEnd(panel);
  const z1 = panelTop(panel);
  const wells: string[] = [];
  const cells = [];
  for (let row = 0; row < panel.rows; row++) {
    for (let column = 0; column < panel.cols; column++) {
      const u = panel.x0 + column * (CELL.w + CELL.gap);
      const z = z1 - (row + 1) * CELL.h - row * CELL.gap;
      const cell = row * panel.cols + column;
      wells.push(roundRect(u, z, u + CELL.w, z + CELL.h, 0.8, 2));
      cells.push(
        <rect
          key={cell}
          className="cell"
          data-cell={`${String(index)}-${String(cell)}`}
          x={u + 0.9}
          y={-(z + CELL.h - 0.9)}
          width={CELL.w - 1.8}
          height={CELL.h - 1.8}
          rx={0.5}
          opacity={0}
        />,
      );
    }
  }
  return mark(
    <g data-panel={index}>
      <path
        className="iso-line"
        data-tone="lo"
        d={roundRect(panel.x0 - 2, CELL.z0 - 2, x1 + 2, z1 + 2, 1.4)}
      />
      <path className="cell-well" d={wells.join("")} />
      {cells}
      <path className="iso-line" data-tone="lo" d={wells.join("")} />
      <g data-drops={index} />
    </g>,
  );
}

const BACK = "downloading.loom.back";

function rails(P: Projection): Drawn[] {
  return PANELS.flatMap((panel, index) => {
    const rail: Plan = {
      x: panel.x0,
      y: LOOM.y + LOOM.d - 5,
      w: panelEnd(panel) - panel.x0,
      d: 3,
      r: 0.8,
    };
    const slider: Plan = { x: panel.x0 - 1.4, y: rail.y - 0.6, w: 2.8, d: 4.2, r: 0.6 };
    return [
      owned(
        box(P, `downloading.loom.rail.${String(index)}`, rail, 6, 0.6, {
          tone: "lo",
          steps: 2,
          bevel: 0,
          crease: "none",
        }),
        BACK,
      ),
      group({ "data-progress": String(index) }, [
        owned(
          box(P, `downloading.loom.slider.${String(index)}`, slider, 6.6, 1.6, {
            tone: "hi",
            steps: 2,
            bevel: 0,
            crease: "lo",
          }),
          BACK,
        ),
      ]),
    ];
  });
}

function frame(P: Projection): Drawn[] {
  return [
    halo(P, LOOM, 0, 1, 6),
    box(P, "downloading.loom.base", LOOM, 0, 6, { bevel: 0.6 }),
    ...screws(
      P,
      [
        [LOOM.x + 4, LOOM.y + LOOM.d - 4],
        [LOOM.x + LOOM.w - 4, LOOM.y + LOOM.d - 4],
      ],
      6,
    ),
    box(P, BACK, { x: LOOM.x + 6, y: -104, w: LOOM.w - 12, d: 14, r: 1 }, 6, BEAM_Z - 6, {
      bevel: 0.4,
    }),
    ...[LOOM.x, LOOM.x + LOOM.w - 7].flatMap((px, index) => [
      owned(
        box(
          P,
          `downloading.loom.post.${String(index)}`,
          { x: px, y: -105, w: 7, d: 16, r: 1 },
          6,
          BEAM_Z - 6,
          { bevel: 0.5 },
        ),
        BACK,
      ),
      dots([iso([px + 3.5, -89, 14], P), iso([px + 3.5, -89, BEAM_Z - 8], P)], 0.42),
    ]),
    mark(
      <g transform={faceY(P, BOARD_Y)}>
        {PANELS.map((panel, index) => (
          <g key={panel.x0}>{panelFace(panel, index).el}</g>
        ))}
        <path className="iso-line" data-tone="faint" d={roundRect(-64.8, 10, -63.2, 84, 0.8)} />
      </g>,
    ),
  ];
}

function beam(P: Projection): Drawn[] {
  const hopper = frameOf(
    [HOPPER.x, HOPPER.y + HOPPER.d / 2, BEAM_Z + 6],
    [1, 0, 0],
    [0, 1, 0],
    [0, 0, 1],
  );
  const mouth: Plane[] = [
    [HOPPER.x + 1.5, HOPPER.y - 1.2],
    [HOPPER.x + HOPPER.w - 1.5, HOPPER.y - 1.2],
    [HOPPER.x + HOPPER.w - 1.5, HOPPER.y + HOPPER.d + 1.2],
    [HOPPER.x + 1.5, HOPPER.y + HOPPER.d + 1.2],
  ];
  const sides: Plane[] = [
    [-5, 0],
    [5, 0],
    [7.5, 9],
    [-7.5, 9],
  ];
  return [
    box(
      P,
      "downloading.loom.beam",
      { x: LOOM.x - 2, y: -107, w: LOOM.w + 4, d: 18, r: 1.5 },
      BEAM_Z,
      6,
      { bevel: 0.6 },
    ),
    line(lineOnTop([LOOM.x - 2, -93], [LOOM.x + LOOM.w + 2, -93], BEAM_Z + 6, P), "lo"),
    ...screws(
      P,
      [
        [LOOM.x + 2, -98],
        [LOOM.x + LOOM.w - 2, -98],
      ],
      BEAM_Z + 6,
    ),
    prism(P, "downloading.loom.hopper", hopper, sides, 0, HOPPER.w),
    line(onTop(mouth, HOPPER_TOP, P, true), "lo"),
  ];
}

function shuttle(P: Projection): Drawn {
  const body: Plan = { x: SHUTTLE_HOME - 6, y: -95, w: 12, d: 6, r: 1 };
  return group({ "data-shuttle": "" }, [
    owned(
      box(P, "downloading.loom.shuttle", body, BEAM_Z - 6, 5, {
        tone: "hi",
        steps: 2,
        bevel: 0.4,
        crease: "lo",
      }),
      BACK,
    ),
    dots([iso([SHUTTLE_HOME, -89, BEAM_Z - 3.5], P)], 0.55, "hi"),
  ]);
}

function loom(P: Projection): Item {
  return item(
    "downloading.loom",
    [LOOM.x + LOOM.w / 2, LOOM.y + LOOM.d / 2, 0],
    [...frame(P), ...beam(P), shuttle(P), ...rails(P)],
  );
}

function chutes(P: Projection): Item {
  const chute = tube(P, "downloading.chute", CHUTE_IN, CHUTE_R, { className: "glass" });
  return item(
    "downloading.chute",
    [(CHUTE_IN[0]?.[0] ?? 0) / 2 + (CHUTE_IN[1]?.[0] ?? 0) / 2, -70, 100],
    [chute, mark(<g data-pieces="" />)],
    [chute.pipe],
  );
}

/** Downloading's machine, as the items it sorts by. */
export const board = (P: Projection): Item[] => [antenna(P), loom(P), chutes(P)];
