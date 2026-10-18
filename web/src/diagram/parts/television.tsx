import {
  type Plan,
  type Point3,
  type Projection,
  corners,
  insetPlan,
  iso,
  pathOf,
  roundedPlan,
  sideSeam,
} from "@anatomy/iso-kit";
import { fillet, frameAlong } from "@anatomy/lathe";

import {
  type Drawn,
  type Item,
  box,
  disc,
  dots,
  group,
  halo,
  item,
  line,
  mark,
  post,
} from "./part";
import { faceY, roundRect } from "./face";
import { tube } from "./tube";

// Media Server link's machine: a small television on a stand, its cable run
// down into the tile. Its screen stays dark until the Media Server link
// reports whether the Media Server answers.

const FOOT: Plan = { x: -10, y: -7, w: 20, d: 14, r: 2 };
const NECK_TOP = 9;
const CASE: Plan = { x: -14, y: -8, w: 28, d: 17, r: 2.4 };
const CASE_Z = NECK_TOP + 1.2;
const CASE_TOP = CASE_Z + 21;
const BACK: Plan = { x: -10, y: -14, w: 20, d: 6.4, r: 2.4 };
const FRONT = CASE.y + CASE.d;
const SCREEN = { u0: -12, u1: 5, z0: CASE_Z + 3, z1: CASE_TOP - 3 } as const;
const GROMMET: [number, number] = [0, -20];

function face(P: Projection): Drawn {
  const grille: string[] = [];
  for (let index = 0; index < 4; index++) {
    grille.push(roundRect(7.6, CASE_Z + 3 + index * 2.2, 12, CASE_Z + 4 + index * 2.2, 0.5, 2));
  }
  return mark(
    <g transform={faceY(P, FRONT)}>
      <path
        className="iso-line"
        data-tone="lo"
        d={roundRect(SCREEN.u0 - 1, SCREEN.z0 - 1, SCREEN.u1 + 1, SCREEN.z1 + 1, 2.6)}
      />
      <path
        className="screen"
        data-screen=""
        d={roundRect(SCREEN.u0, SCREEN.z0, SCREEN.u1, SCREEN.z1, 2.2)}
      />
      <path className="iso-line" data-tone="faint" d={grille.join("")} />
      {[CASE_TOP - 4.6, CASE_TOP - 9.4].map((z) => (
        <g key={z}>
          <circle className="hub" cx={9.8} cy={-z} r={1.7} />
          <path className="iso-line" data-tone="hi" d={`M9.8 ${String(-z)}v-1.7`} />
        </g>
      ))}
    </g>,
  );
}

function ears(P: Projection): Drawn[] {
  return (
    [
      [-0.55, 0.25],
      [0.5, 0.35],
    ] as const
  ).flatMap(([lean, back], index) => {
    const frame = frameAlong([lean * 2, -2, CASE_TOP + 2.2], [lean, -back, 1]);
    const name = `media-server-link.ear.${String(index)}`;
    return [
      disc(P, name, frame, 0, 15, 0.45, { tone: "mid", crease: "none" }),
      disc(P, `${name}.tip`, frame, 15, 16.4, 0.9, { tone: "hi", crease: "none" }),
    ];
  });
}

function cable(P: Projection): ReturnType<typeof tube> {
  const route: Point3[] = [
    [4, BACK.y, CASE_Z + 6],
    [4, GROMMET[1] + 2, CASE_Z + 6],
    [4, GROMMET[1] + 2, 3],
    [GROMMET[0], GROMMET[1], 3],
    [GROMMET[0], GROMMET[1], 1.6],
  ];
  return tube(P, "media-server-link.cable", fillet(route, 2.4, 6), 0.8, { tone: "lo" });
}

/** Slots on a side face at x, from u0 along -y, each `w` wide and `h` tall from z. */
function slots(
  P: Projection,
  x: number,
  {
    u0,
    count,
    pitch,
    w,
    h,
    z,
  }: { u0: number; count: number; pitch: number; w: number; h: number; z: number },
): Drawn {
  const paths = Array.from({ length: count }, (_, index) =>
    pathOf(
      roundedPlan({ x: u0 + index * pitch, y: z, w, d: h, r: w / 2 }, 2).map(([u, pz]) =>
        iso([x, u, pz], P),
      ),
      true,
    ),
  );
  return line(paths.join(""), "lo");
}

function stand(P: Projection): Drawn[] {
  return [
    halo(P, FOOT, 0, 2, 5),
    box(P, "media-server-link.foot", FOOT, 0, 2, { tone: "lo", bevel: 0.4 }),
    post(P, "media-server-link.grommet", GROMMET, 2.4, 0, 1.6, { tone: "lo", steps: 20 }),
    post(P, "media-server-link.neck", [0, 0], 2.2, 2, NECK_TOP - 2, { steps: 20 }),
    post(P, "media-server-link.swivel", [0, 0], 4, NECK_TOP, 1.2, { tone: "lo", steps: 24 }),
  ];
}

export function television(P: Projection): Item {
  const lead = cable(P);
  return item(
    "media-server-link",
    [0, 0, 0],
    [
      ...stand(P),
      lead,
      box(P, "media-server-link.back", BACK, CASE_Z + 2, CASE_TOP - CASE_Z - 4, {
        bevel: 0.6,
        inner: line(sideSeam(BACK, CASE_Z + 10, P, 3), "faint").el,
      }),
      slots(P, BACK.x + BACK.w, {
        u0: BACK.y + 1.2,
        count: 4,
        pitch: 1.4,
        w: 0.7,
        h: 4,
        z: CASE_TOP - 9,
      }),
      box(P, "media-server-link.case", CASE, CASE_Z, CASE_TOP - CASE_Z, {
        bevel: 0.8,
        inner: line(sideSeam(CASE, CASE_Z + 2.2, P, 3), "faint").el,
      }),
      face(P),
      slots(P, CASE.x + CASE.w, {
        u0: CASE.y + 3,
        count: 5,
        pitch: 2.4,
        w: 1,
        h: 5,
        z: CASE_TOP - 8,
      }),
      group({ "data-led": "standby", className: "led" }, [
        dots([iso([9.8, FRONT, CASE_Z + 2.2], P)], 0.6, "hi"),
      ]),
      dots(
        corners(insetPlan(CASE, 2), 0).map(([x, y]) => iso([x, y, CASE_TOP], P)),
        0.38,
      ),
      post(P, "media-server-link.mount", [0, -2], 2.4, CASE_TOP, 2.2, { steps: 20, bevel: 0.3 }),
      ...ears(P),
    ],
    [lead.pipe],
  );
}
