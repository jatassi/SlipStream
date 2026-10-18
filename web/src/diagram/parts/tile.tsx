import {
  type Plan,
  type Plane,
  type Projection,
  corners,
  insetPlan,
  iso,
  lineOnTop,
  pathOf,
  planOutline,
  roundedPlan,
  sideArc,
  sideMatrix,
  sideSeam,
  topMatrix,
} from "@anatomy/iso-kit";
import { FaceText } from "@anatomy/react/draw";

import { FEET, SLAB, TILE, TOP } from "../layout";
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
  post,
  screws,
} from "./part";
import { roundRect } from "./face";

const FOOT_INSET = 22;

/** Where the tile's feet stand: its corners, and the middle of its long edges. */
export const FEET_AT: readonly Plane[] = [
  ...corners(TILE, FOOT_INSET),
  [0, TILE.y + FOOT_INSET],
  [0, TILE.y + TILE.d - FOOT_INSET],
];

const byDepth = (points: readonly Plane[]): Plane[] =>
  points.toSorted(([ax, ay], [bx, by]) => ax + ay - (bx + by));

function feet(P: Projection): Drawn[] {
  return byDepth(FEET_AT).flatMap((at, index) => [
    post(P, `stand.foot.${String(index)}.pad`, at, 8, 0, 1.8, { tone: "lo", steps: 28 }),
    post(P, `tile.foot.${String(index)}`, at, 6, 1.8, FEET - 1.8, { steps: 28 }),
    line(sideArc(at[0], at[1], 6, 5, P, 16), "faint"),
  ]);
}

function mat(P: Projection): Drawn {
  const inner = insetPlan(TILE, 6.6);
  const lines: string[] = [];
  for (let x = TILE.x + 40; x < TILE.x + TILE.w - 20; x += 40) {
    lines.push(lineOnTop([x, inner.y], [x, inner.y + inner.d], TOP, P));
  }
  for (let y = TILE.y + 40; y < TILE.y + TILE.d - 20; y += 40) {
    lines.push(lineOnTop([inner.x, y], [inner.x + inner.w, y], TOP, P));
  }
  return line(lines.join(""), "faint");
}

function vents(P: Projection): Drawn {
  const right = TILE.x + TILE.w;
  const slots: string[] = [];
  for (let index = 0; index < 10; index++) {
    const y0 = TILE.y + 20 + index * 6;
    slots.push(
      pathOf(
        roundedPlan({ x: y0, y: FEET + 4, w: 2.2, d: 6, r: 1 }, 3).map(([u, z]) =>
          iso([right, u, z], P),
        ),
        true,
      ),
    );
  }
  return line(slots.join(""), "lo");
}

/** The tile itself: feet, the slab with its seam, the inlaid grooves, the floor grid and the corner screws. */
export function base(P: Projection): Item {
  const seam = line(sideSeam(TILE, FEET + SLAB * 0.38, P, 8), "faint");
  return item(
    "tile.base",
    [0, 0, FEET],
    [
      halo(P, TILE, 0, TOP, 12),
      ...feet(P),
      box(P, "tile.slab", TILE, FEET, SLAB, { steps: 8, bevel: 1.6, inner: seam.el }),
      line(planOutline(insetPlan(TILE, 5), TOP, P), "lo"),
      line(planOutline(insetPlan(TILE, 6.6), TOP, P), "faint"),
      mat(P),
      vents(P),
      ...screws(P, corners(TILE, 11), TOP),
    ],
  );
}

/** The plaque on the tile's front-right face: the Node's name, engraved, and what it is doing. */
const PLAQUE = { from: -36, to: 110, z0: FEET + 1.4, z1: TOP - 1.4 };
const PLAQUE_FACE = TILE.x + TILE.w;

export function plaque(P: Projection, name: string, state: string): Item {
  const face = sideMatrix([PLAQUE_FACE + 1, 0], 0, P, "right");
  const plan: Plan = { x: PLAQUE_FACE, y: PLAQUE.from, w: 1, d: PLAQUE.to - PLAQUE.from, r: 0.4 };
  const u0 = -PLAQUE.to;
  const u1 = -PLAQUE.from;
  return item(
    "tile.plaque",
    [PLAQUE_FACE, (PLAQUE.from + PLAQUE.to) / 2, FEET + SLAB / 2],
    [
      box(P, "tile.plaque", plan, PLAQUE.z0, PLAQUE.z1 - PLAQUE.z0, {
        tone: "lo",
        crease: "none",
        bevel: 0,
      }),
      mark(
        <g transform={face}>
          <path
            className="iso-line"
            data-tone="faint"
            d={roundRect(u0 + 1.4, PLAQUE.z0 + 1, u1 - 1.4, PLAQUE.z1 - 1, 0.8)}
          />
          <FaceText
            transform={`translate(${String(u0 + 4)} ${String(-(PLAQUE.z0 + 2.8))})`}
            size={7.6}
            tone="hi"
          >
            {name}
          </FaceText>
          <FaceText
            transform={`translate(${String(u1 - 4)} ${String(-(PLAQUE.z0 + 2.8))})`}
            size={2.8}
            tone="mid"
            anchor="end"
          >
            {state.toUpperCase()}
          </FaceText>
        </g>,
      ),
      dots(
        [
          iso([PLAQUE_FACE + 1, PLAQUE.to - 2.4, PLAQUE.z0 + 2], P),
          iso([PLAQUE_FACE + 1, PLAQUE.to - 2.4, PLAQUE.z1 - 2], P),
        ],
        0.4,
      ),
    ],
  );
}

/** The status lamp on the tile's front corner: its glow takes the Node's state. */
const LAMP: Plane = [150, 60];

export function lamp(P: Projection): Item {
  const [x, y] = LAMP;
  const glowAt = iso([x, y, TOP + 8.2], P);
  return item(
    "tile.lamp",
    [x, y, TOP],
    [
      post(P, "tile.lamp.base", LAMP, 4, TOP, 2, { tone: "lo", bevel: 0.3 }),
      post(P, "tile.lamp.neck", LAMP, 2.6, TOP + 2, 3, { bevel: 0.3 }),
      group({ "data-lamp": "" }, [
        post(P, "tile.lamp.lens", LAMP, 2.2, TOP + 5, 3.2, { tone: "hi", steps: 20 }),
        mark(
          <g className="lamp-glow">
            <circle className="gh" cx={glowAt[0]} cy={glowAt[1]} r={2.2 * P.scale} />
            <circle className="gc" cx={glowAt[0]} cy={glowAt[1]} r={0.8 * P.scale} />
          </g>,
        ),
      ]),
      dots([iso([x - 3.2, y + 0.6, TOP + 2], P), iso([x + 3.2, y - 0.6, TOP + 2], P)], 0.38),
    ],
  );
}

/** The places Role chips take on the tile's top, along its front edge, two rows of five. */
export const CHIP = { w: 56, d: 14, h: 2.4, gap: 5, rows: [94, 77] as const, from: -148 };
export const CHIP_SLOTS = 10;

export function chipPlan(slot: number): Plan {
  const row = Math.floor(slot / 5);
  const column = slot % 5;
  const y = CHIP.rows[row];
  if (y === undefined) {
    throw new Error(`no chip slot ${String(slot)}`);
  }
  return { x: CHIP.from + column * (CHIP.w + CHIP.gap), y, w: CHIP.w, d: CHIP.d, r: 1.6 };
}

/** A Role chip: a key cap on the tile's top with the Role's name engraved and what it is doing under it. */
export function chip(P: Projection, slot: number, label: string, note: string): Item {
  const plan = chipPlan(slot);
  const z = TOP + CHIP.h;
  const lit: Plane = [plan.x + 4.6, plan.y + plan.d / 2];
  return item(
    `chip.${String(slot)}`,
    [plan.x + plan.w / 2, plan.y + plan.d / 2, TOP],
    [
      box(P, `chip.${String(slot)}.cap`, plan, TOP, CHIP.h, {
        tone: "hi",
        crease: "lo",
        bevel: 0.4,
      }),
      line(planOutline(insetPlan(plan, 1.2), z, P, 3), "faint"),
      mark(
        <>
          <FaceText
            transform={topMatrix([plan.x + 8.6, plan.y + 7.4], z, P, "x")}
            size={4.4}
            tone="hi"
          >
            {label}
          </FaceText>
          {note !== "" && (
            <FaceText
              transform={topMatrix([plan.x + 8.6, plan.y + 11.2], z, P, "x")}
              size={2.4}
              tone="mid"
            >
              {note.toUpperCase()}
            </FaceText>
          )}
        </>,
      ),
      group({ "data-chip-led": "" }, [dots([iso([lit[0], lit[1], z], P)], 0.9, "hi")]),
    ],
  );
}
