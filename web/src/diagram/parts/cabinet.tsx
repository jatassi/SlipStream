import {
  type Plan,
  type Point3,
  type Projection,
  corners,
  insetPlan,
  iso,
  pathOf,
  planOutline,
  roundedPlan,
  sideSeam,
} from "@anatomy/iso-kit";
import { hullOf } from "@anatomy/lathe";
import { useId } from "react";

import { box, dots, group, halo, item, line, mark, screws, type Drawn, type Item } from "./part";
import { faceY, roundRect } from "./face";
import { wiring } from "./lead";

// Storage's machine: a steel cabinet of four drawers. It stands still until
// S16 (Library roots) gives it live figures. Its cable leaves under the
// bottom drawer's lip, so the cabinet draws it between its body and drawers.

const CAB: Plan = { x: 40, y: -114, w: 84, d: 50, r: 3 };
const FRONT = CAB.y + CAB.d;
const TOP = 100;
const DRAWER_H = 21;
const DRAWERS = 4;

const drawerZ = (index: number): number => 7 + index * 23;

function ClippedDrawer({ hull, children }: { hull: string; children: React.ReactNode }) {
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

function hullOfDrawer(P: Projection, z0: number): string {
  const near: Point3[] = [
    [CAB.x + 2, FRONT - 0.2, z0 - 1],
    [CAB.x + CAB.w - 2, FRONT - 0.2, z0 - 1],
    [CAB.x + CAB.w - 2, FRONT - 0.2, z0 + DRAWER_H + 1],
    [CAB.x + 2, FRONT - 0.2, z0 + DRAWER_H + 1],
  ];
  const far = near.map(([x, y, z]): Point3 => [x, y + 30, z]);
  return pathOf(hullOf([...near, ...far].map((point) => iso(point, P))), true);
}

const GAUGE = { from: CAB.x + 50, to: CAB.x + CAB.w - 10 };

function drawerFace(P: Projection, z0: number, fy: number): Drawn {
  const { from, to } = GAUGE;
  const needle = from + (to - from) * 0.9;
  return mark(
    <g transform={faceY(P, fy)}>
      <path
        className="iso-line"
        data-tone="mid"
        d={roundRect(CAB.x + 8, z0 + 9, CAB.x + 26, z0 + 16, 0.8)}
      />
      <path className="card" d={roundRect(CAB.x + 9.2, z0 + 10.2, CAB.x + 24.8, z0 + 14.8, 0.4)} />
      <path className="iso-line" data-tone="lo" d={roundRect(from, z0 + 4, to, z0 + 7.4, 1.2)} />
      <path
        className="iso-line"
        data-tone="mid"
        d={roundRect(needle - 0.3, z0 + 3.4, needle + 0.3, z0 + 8, 0.3)}
      />
    </g>,
  );
}

function drawer(P: Projection, index: number): Drawn {
  const z0 = drawerZ(index);
  const fy = FRONT + 2.2;
  const px = CAB.x + 36;
  const name = `storage.drawer.${String(index)}`;
  const parts: Drawn[] = [
    box(
      P,
      `${name}.front`,
      { x: CAB.x + 3.5, y: FRONT, w: CAB.w - 7, d: 2.2, r: 0.8 },
      z0,
      DRAWER_H,
      {
        tone: "hi",
        crease: "lo",
        bevel: 0.5,
      },
    ),
    drawerFace(P, z0, fy),
    group({ "data-drive-led": String(index), className: "led" }, [
      dots([iso([CAB.x + CAB.w - 7, fy, z0 + 15], P)], 0.9, "hi"),
    ]),
    dots([iso([CAB.x + 6, fy, z0 + 3], P), iso([CAB.x + CAB.w - 6, fy, z0 + 3], P)], 0.38),
    box(P, `${name}.post.0`, { x: px - 7, y: fy, w: 2.4, d: 3, r: 0.6 }, z0 + 11, 2.4, {
      steps: 2,
      bevel: 0,
      crease: "none",
    }),
    box(P, `${name}.post.1`, { x: px + 4.6, y: fy, w: 2.4, d: 3, r: 0.6 }, z0 + 11, 2.4, {
      steps: 2,
      bevel: 0,
      crease: "none",
    }),
    box(P, `${name}.handle`, { x: px - 8, y: fy + 3, w: 16, d: 2, r: 0.8 }, z0 + 10.6, 3.2, {
      tone: "hi",
      steps: 2,
      bevel: 0.3,
      crease: "lo",
    }),
  ];
  return {
    el: (
      <ClippedDrawer hull={hullOfDrawer(P, z0)}>
        {group({ "data-drawer": String(index) }, parts).el}
      </ClippedDrawer>
    ),
    shapes: parts.flatMap(({ shapes }) => shapes),
  };
}

export function cabinet(P: Projection): Item {
  const lead = wiring(P, "storage.lead", {
    from: [CAB.x + 12, FRONT, 5],
    toward: [0, 1],
    reach: 10,
  });
  const vents: string[] = [];
  for (let index = 0; index < 7; index++) {
    const y0 = CAB.y + 8 + index * 4.6;
    vents.push(
      pathOf(
        roundedPlan({ x: y0, y: 78, w: 2, d: 14, r: 1 }, 3).map(([u, z]) =>
          iso([CAB.x + CAB.w, u, z], P),
        ),
        true,
      ),
    );
  }
  return item(
    "storage",
    [CAB.x + CAB.w / 2, CAB.y + CAB.d / 2, 0],
    [
      halo(P, CAB, 0, 4, 7),
      box(P, "storage.plinth", insetPlan(CAB, 2.5), 0, 4, { tone: "lo", bevel: 0 }),
      box(P, "storage.body", CAB, 4, TOP - 4, {
        bevel: 0.8,
        inner: line(sideSeam(CAB, 50, P, 3), "faint").el,
      }),
      line(vents.join(""), "lo"),
      box(P, "storage.lid", insetPlan(CAB, -1.2), TOP, 3, { bevel: 0.8 }),
      line(planOutline(insetPlan(CAB, 4), TOP + 3, P, 3), "faint"),
      ...screws(P, corners(CAB, 3), TOP + 3),
      mark(
        <g transform={faceY(P, FRONT)}>
          <path
            className="iso-line"
            data-tone="faint"
            d={roundRect(CAB.x + 2, 5, CAB.x + CAB.w - 2, TOP - 2, 1.5)}
          />
        </g>,
      ),
      ...lead.parts,
      ...Array.from({ length: DRAWERS }, (_, index) => drawer(P, index)),
    ],
    [lead.pipe],
  );
}
