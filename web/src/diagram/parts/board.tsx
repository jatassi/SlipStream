import { type Plan, type Plane, type Point3, type Projection, iso, lineOnTop, onTop, sideArc } from "@anatomy/iso-kit";
import { dotsOf, fillet, frameAlong, frameOf } from "@anatomy/lathe";

import {
  type Drawn,
  type Item,
  box,
  dots,
  faceY,
  group,
  halo,
  owned,
  item,
  line,
  mark,
  post,
  prism,
  roundRect,
  screws,
} from "./part";
import { tube } from "./tube";

// Downloading's machine: an antenna takes pieces in, two chutes carry them to
// a hopper on a loom, and the loom's shuttle drops each into the board of the
// torrent it belongs to. Coordinates are on the tile's top, at z = 0.

const ANT: Plane = [-150, -60];
const ANT_TOP = 146;
const BOOM = { from: -178, to: -142 };
const RX = { x: -158, y: -68, w: 16, d: 14, r: 1.5, z: 100, h: 16 };

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
export function cellCentre(panel: Panel, index: number): { readonly u: number; readonly z: number } {
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

function antenna(P: Projection): Item {
  const [x, y] = ANT;
  const elements = [-174, -167, -160, -153, -146].map((ex, index) => ({ x: ex, half: 7 + index * 1.5 }));
  const boom: Plan = { x: BOOM.from, y: y - 1.2, w: BOOM.to - BOOM.from, d: 2.4, r: 0.6 };
  const element = (side: "back" | "front", e: { x: number; half: number }, index: number): Drawn =>
    box(
      P,
      `downloading.antenna.${side}.${String(index)}`,
      side === "back"
        ? { x: e.x - 0.6, y: y - e.half, w: 1.2, d: e.half - 1.2, r: 0.5 }
        : { x: e.x - 0.6, y: y + 1.2, w: 1.2, d: e.half - 1.2, r: 0.5 },
      ANT_TOP + 0.2,
      1.2,
      { steps: 2, bevel: 0, crease: "none" },
    );
  return item("downloading.antenna", [x, y, 0], [
    halo(P, { x: x - 12, y: y - 12, w: 24, d: 24, r: 12 }, 0, 1, 5),
    post(P, "downloading.antenna.foot", ANT, 11, 0, 2, { tone: "lo", steps: 40, bevel: 0.5 }),
    dots(dotsOf(frameAlong([x, y, 0], [0, 0, 1]), 2, 8.4, 6, P, { all: true }), 0.45),
    post(P, "downloading.antenna.collar", ANT, 5, 2, 5, { steps: 28, bevel: 0.4 }),
    post(P, "downloading.antenna.mast", ANT, 2.6, 7, RX.z - 7, {
      steps: 20,
      crease: "none",
      inner: line([30, 60].map((z) => sideArc(x, y, 2.6, z, P, 12)).join(""), "faint").el,
    }),
    ...[40, 70].map((z) =>
      owned(
        post(P, `downloading.antenna.band.${String(z)}`, ANT, 3.6, z - 1.5, 3, { steps: 20, bevel: 0.3 }),
        "downloading.antenna.mast",
      ),
    ),
    box(P, "downloading.antenna.receiver", RX, RX.z, RX.h, { bevel: 0.6 }),
    mark(
      <g transform={faceY(P, RX.y + RX.d)}>
        <path className="iso-line" data-tone="faint" d={roundRect(RX.x + 2.4, RX.z + 3, RX.x + RX.w - 2.4, RX.z + RX.h - 3, 1.2)} />
        <path className="iso-line" data-tone="lo" d={roundRect(RX.x + 3.4, RX.z + 9.5, RX.x + 9.5, RX.z + 11.8, 1)} />
      </g>,
    ),
    group({ "data-led": "rx" }, [dots([iso([RX.x + RX.w - 4, RX.y + RX.d, RX.z + 10.6], P)], 0.85, "hi")]),
    dots(
      [iso([RX.x + 2.2, RX.y + RX.d, RX.z + 1.8], P), iso([RX.x + RX.w - 2.2, RX.y + RX.d, RX.z + 1.8], P)],
      0.4,
    ),
    post(P, "downloading.antenna.upper", ANT, 2.2, RX.z + RX.h, ANT_TOP - RX.z - RX.h - 3, { steps: 18, crease: "none" }),
    post(P, "downloading.antenna.head", ANT, 3.4, ANT_TOP - 3, 3, { steps: 18, bevel: 0.3 }),
    ...elements.map((e, index) => element("back", e, index)),
    box(P, "downloading.antenna.boom", boom, ANT_TOP, 2.4, { steps: 2, bevel: 0 }),
    ...elements.map((e, index) => element("front", e, index)),
    dots(
      elements.map((e) => iso([e.x, y, ANT_TOP + 2.4], P)),
      0.4,
    ),
    post(P, "downloading.antenna.whip", ANT, 1, ANT_TOP + 2.4, 10, { steps: 12, crease: "none" }),
    group({ "data-led": "beacon" }, [post(P, "downloading.antenna.beacon", ANT, 2, ANT_TOP + 12.4, 3, { tone: "hi", steps: 16, lit: true })]),
  ]);
}

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
        />,
      );
    }
  }
  return mark(
    <g data-panel={index}>
      <path className="iso-line" data-tone="lo" d={roundRect(panel.x0 - 2, CELL.z0 - 2, x1 + 2, z1 + 2, 1.4)} />
      <path className="cell-well" d={wells.join("")} />
      {cells}
      <path className="iso-line" data-tone="lo" d={wells.join("")} />
      <g data-drops={index} />
    </g>,
  );
}

function loom(P: Projection): Item {
  const hopper = frameOf([HOPPER.x, HOPPER.y + HOPPER.d / 2, BEAM_Z + 6], [1, 0, 0], [0, 1, 0], [0, 0, 1]);
  const rails = PANELS.map((panel, index) => {
    const rail: Plan = { x: panel.x0, y: LOOM.y + LOOM.d - 5, w: panelEnd(panel) - panel.x0, d: 3, r: 0.8 };
    return [
      owned(
        box(P, `downloading.loom.rail.${String(index)}`, rail, 6, 0.6, { tone: "lo", steps: 2, bevel: 0, crease: "none" }),
        "downloading.loom.back",
      ),
      group({ "data-progress": String(index) }, [
        owned(
          box(
          P,
          `downloading.loom.slider.${String(index)}`,
          { x: panel.x0 - 1.4, y: rail.y - 0.6, w: 2.8, d: 4.2, r: 0.6 },
          6.6,
          1.6,
          { tone: "hi", steps: 2, bevel: 0, crease: "lo" },
          ),
          "downloading.loom.back",
        ),
      ]),
    ];
  });
  return item("downloading.loom", [LOOM.x + LOOM.w / 2, LOOM.y + LOOM.d / 2, 0], [
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
    box(P, "downloading.loom.back", { x: LOOM.x + 6, y: -104, w: LOOM.w - 12, d: 14, r: 1 }, 6, BEAM_Z - 6, { bevel: 0.4 }),
    ...[LOOM.x, LOOM.x + LOOM.w - 7].flatMap((px, index) => [
      owned(
        box(P, `downloading.loom.post.${String(index)}`, { x: px, y: -105, w: 7, d: 16, r: 1 }, 6, BEAM_Z - 6, { bevel: 0.5 }),
        "downloading.loom.back",
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
    box(P, "downloading.loom.beam", { x: LOOM.x - 2, y: -107, w: LOOM.w + 4, d: 18, r: 1.5 }, BEAM_Z, 6, { bevel: 0.6 }),
    line(lineOnTop([LOOM.x - 2, -93], [LOOM.x + LOOM.w + 2, -93], BEAM_Z + 6, P), "lo"),
    ...screws(
      P,
      [
        [LOOM.x + 2, -98],
        [LOOM.x + LOOM.w - 2, -98],
      ],
      BEAM_Z + 6,
    ),
    prism(
      P,
      "downloading.loom.hopper",
      hopper,
      [
        [-5, 0],
        [5, 0],
        [7.5, 9],
        [-7.5, 9],
      ],
      0,
      HOPPER.w,
    ),
    line(
      onTop(
        [
          [HOPPER.x + 1.5, HOPPER.y - 1.2],
          [HOPPER.x + HOPPER.w - 1.5, HOPPER.y - 1.2],
          [HOPPER.x + HOPPER.w - 1.5, HOPPER.y + HOPPER.d + 1.2],
          [HOPPER.x + 1.5, HOPPER.y + HOPPER.d + 1.2],
        ],
        BEAM_Z + 15,
        P,
        true,
      ),
      "lo",
    ),
    group({ "data-shuttle": "" }, [
      owned(
      box(P, "downloading.loom.shuttle", { x: SHUTTLE_HOME - 6, y: -95, w: 12, d: 6, r: 1 }, BEAM_Z - 6, 5, {
        tone: "hi",
        steps: 2,
        bevel: 0.4,
        crease: "lo",
      }),
        "downloading.loom.back",
      ),
      dots([iso([SHUTTLE_HOME, -89, BEAM_Z - 3.5], P)], 0.55, "hi"),
    ]),
    ...rails.flat(),
  ]);
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
