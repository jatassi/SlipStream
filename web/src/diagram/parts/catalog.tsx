import {
  type Plan,
  type Point3,
  type Projection,
  corners,
  insetPlan,
  iso,
  pathOf,
  planOutline,
  sideSeam,
} from "@anatomy/iso-kit";
import { hullOf } from "@anatomy/lathe";
import { type ReactNode, useId } from "react";

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
  post,
  screws,
} from "./part";
import { faceY, roundRect } from "./face";

// Metadata's machine: a card catalog. The live loop slides one drawer out
// and back each time a Work's artwork and details are fetched.

const CAT: Plan = { x: -17, y: -12, w: 34, d: 24, r: 1.5 };
const FRONT = CAT.y + CAT.d;
const BODY_Z = 2.6;
const BODY_TOP = 29;
const DRAWER = { w: 14.6, h: 7.4, gap: 1, from: CAT.x + 1.9, z0: BODY_Z + 2.4 } as const;
const COLUMNS = 2;
const ROWS = 3;
/** The drawer the live loop slides, as its column and row. */
const LIVE = { column: 1, row: 1 } as const;
/** How far the live drawer slides out. */
export const DRAWER_TRAVEL = 9;

const drawerX = (column: number): number => DRAWER.from + column * (DRAWER.w + DRAWER.gap);
const drawerZ = (row: number): number => DRAWER.z0 + row * (DRAWER.h + DRAWER.gap);

function drawerFront(P: Projection, name: string, x: number, z: number): Drawn[] {
  const fy = FRONT + 1.4;
  return [
    box(P, `${name}.front`, { x, y: FRONT, w: DRAWER.w, d: 1.4, r: 0.5 }, z, DRAWER.h, {
      tone: "hi",
      crease: "lo",
      bevel: 0.3,
    }),
    mark(
      <g transform={faceY(P, fy)}>
        <path
          className="iso-line"
          data-tone="mid"
          d={roundRect(x + 3.6, z + 3.8, x + DRAWER.w - 3.6, z + 6.4, 0.4)}
        />
        <path className="card" d={roundRect(x + 4.2, z + 4.3, x + DRAWER.w - 4.2, z + 5.9, 0.3)} />
      </g>,
    ),
    post(P, `${name}.pull`, [x + DRAWER.w / 2, fy + 0.9], 0.9, z + 1.2, 1.6, {
      tone: "hi",
      steps: 12,
      crease: "none",
    }),
  ];
}

function ClippedToFront({ hull, children }: { hull: string; children: ReactNode }) {
  const id = useId();
  return (
    <>
      <clipPath id={id}>
        <path d={hull} />
      </clipPath>
      <g clipPath={`url(#${id})`}>{children}</g>
    </>
  );
}

/** The space in front of the live drawer's opening, where the drawer shows as it slides out. */
function frontHull(P: Projection, x: number, z: number): string {
  const near: Point3[] = [
    [x - 0.4, FRONT - 0.1, z - 0.4],
    [x + DRAWER.w + 0.4, FRONT - 0.1, z - 0.4],
    [x + DRAWER.w + 0.4, FRONT - 0.1, z + DRAWER.h + 0.2],
    [x - 0.4, FRONT - 0.1, z + DRAWER.h + 0.2],
  ];
  const far = near.map(([px, py, pz]): Point3 => [px, py + DRAWER_TRAVEL + 4, pz]);
  return pathOf(hullOf([...near, ...far].map((point) => iso(point, P))), true);
}

function liveDrawer(P: Projection): Drawn {
  const x = drawerX(LIVE.column);
  const z = drawerZ(LIVE.row);
  const parts: Drawn[] = [
    owned(
      box(
        P,
        "metadata.drawer.box",
        { x: x + 0.6, y: FRONT - 14, w: DRAWER.w - 1.2, d: 14, r: 0.4 },
        z + 0.4,
        DRAWER.h - 1.6,
        { tone: "mid", steps: 2, bevel: 0 },
      ),
      "metadata.body",
    ),
    ...[3.2, 6.4].map((inset) =>
      owned(
        box(
          P,
          `metadata.drawer.card.${String(inset)}`,
          { x: x + 1.6, y: FRONT - inset - 0.4, w: DRAWER.w - 3.2, d: 0.4, r: 0.1 },
          z + DRAWER.h - 2.6,
          2,
          { tone: "hi", steps: 1, bevel: 0, crease: "none" },
        ),
        "metadata.body",
      ),
    ),
    ...drawerFront(P, "metadata.drawer.live", x, z),
  ];
  return {
    el: (
      <ClippedToFront hull={frontHull(P, x, z)}>
        {group({ "data-drawer-slide": "" }, parts).el}
      </ClippedToFront>
    ),
    shapes: parts.flatMap(({ shapes }) => shapes),
  };
}

function drawers(P: Projection): Drawn[] {
  const out: Drawn[] = [];
  for (let row = 0; row < ROWS; row++) {
    for (let column = 0; column < COLUMNS; column++) {
      if (row !== LIVE.row || column !== LIVE.column) {
        out.push(
          ...drawerFront(
            P,
            `metadata.drawer.${String(row)}.${String(column)}`,
            drawerX(column),
            drawerZ(row),
          ),
        );
      }
    }
  }
  return out;
}

function tray(P: Projection): Drawn[] {
  const plan: Plan = { x: -9, y: -7, w: 18, d: 11, r: 1 };
  const z = BODY_TOP + 2;
  return [
    box(P, "metadata.tray", plan, z, 2.4, { tone: "lo", bevel: 0.3 }),
    line(planOutline(insetPlan(plan, 1.2), z + 2.4, P, 3), "faint"),
    ...[-4, -1.4, 1.2].map((y, index) =>
      box(
        P,
        `metadata.tray.card.${String(index)}`,
        { x: -7, y, w: 14, d: 0.5, r: 0.1 },
        z + 2.4,
        4.4 - index * 0.8,
        { tone: "hi", steps: 1, bevel: 0, crease: "none" },
      ),
    ),
  ];
}

export function catalog(P: Projection): Item {
  const lid = insetPlan(CAT, -0.9);
  return item(
    "metadata",
    [0, 0, 0],
    [
      halo(P, CAT, 0, 3, 6),
      ...corners(CAT, 3)
        .toSorted(([ax, ay], [bx, by]) => ax + ay - (bx + by))
        .map((at, index) =>
          post(P, `metadata.foot.${String(index)}`, at, 1.8, 0, BODY_Z, { tone: "lo", steps: 16 }),
        ),
      box(P, "metadata.body", CAT, BODY_Z, BODY_TOP - BODY_Z, {
        bevel: 0.6,
        inner: line(sideSeam(CAT, BODY_Z + 3, P, 3), "faint").el,
      }),
      mark(
        <g transform={faceY(P, FRONT)}>
          <path
            className="iso-line"
            data-tone="faint"
            d={roundRect(CAT.x + 1.2, BODY_Z + 1.6, CAT.x + CAT.w - 1.2, BODY_TOP - 1.4, 1)}
          />
        </g>,
      ),
      ...drawers(P),
      liveDrawer(P),
      box(P, "metadata.lid", lid, BODY_TOP, 2, { bevel: 0.5 }),
      ...screws(P, corners(lid, 2), BODY_TOP + 2),
      ...tray(P),
      dots([iso([CAT.x + CAT.w - 2.4, FRONT, BODY_Z + 1.2], P)], 0.38),
    ],
  );
}
