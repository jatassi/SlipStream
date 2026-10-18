import { type Plan, type Plane, type Projection, corners, iso, sideSeam } from "@anatomy/iso-kit";
import { arcOf, frameAlong, frameOf } from "@anatomy/lathe";

import {
  type Drawn,
  type Item,
  box,
  disc,
  dots,
  halo,
  item,
  line,
  mark,
  owned,
  prism,
  screws,
} from "./part";
import { faceY } from "./face";

// Notifying's machine: a mailbox on a post, its flag down. It stands at rest
// until Notifying reports the mail it sends.

const FOOT: Plan = { x: -6, y: -6, w: 12, d: 12, r: 1.2 };
const POST: Plan = { x: -1.8, y: -1.8, w: 3.6, d: 3.6, r: 0.5 };
const POST_TOP = 26;
const SHELF = POST_TOP + 1.4;
const HALF = 5.4;
const WALL = 7;
const LENGTH = 22;
const BOX_Y = -LENGTH / 2;
const FRONT = BOX_Y + LENGTH;

/** The mailbox's end: straight walls under a round roof, in the plane across it. */
const END: Plane[] = [
  [-HALF, 0],
  [HALF, 0],
  [HALF, WALL],
  ...Array.from({ length: 11 }, (_, index): Plane => {
    const angle = (index / 12) * Math.PI + Math.PI / 12;
    return [HALF * Math.cos(angle), WALL + HALF * Math.sin(angle)];
  }),
  [-HALF, WALL],
];

const across = frameOf([0, BOX_Y, SHELF], [0, 1, 0], [1, 0, 0], [0, 0, 1]);

function door(P: Projection): Drawn {
  const arch = Array.from({ length: 13 }, (_, index) => {
    const angle = (index / 12) * Math.PI;
    const r = HALF - 1;
    return `${index === 0 ? "M" : "L"}${(r * Math.cos(angle)).toFixed(2)} ${(-(SHELF + WALL + r * Math.sin(angle))).toFixed(2)}`;
  }).join("");
  return mark(
    <g transform={faceY(P, FRONT)}>
      <path
        className="iso-line"
        data-tone="mid"
        d={`M${String(HALF - 1)} ${String(-(SHELF + 1))}V${String(-(SHELF + WALL))}${arch.replace(/^M/u, "L")}V${String(-(SHELF + 1))}Z`}
      />
    </g>,
  );
}

function flag(P: Projection): Drawn[] {
  const pivot = frameAlong([HALF, BOX_Y + 6, SHELF + WALL - 1.4], [1, 0, 0]);
  const x = HALF + 0.9;
  return [
    owned(
      box(
        P,
        "notifying.flag.arm",
        { x, y: BOX_Y + 5, w: 0.7, d: 13, r: 0.3 },
        SHELF + WALL - 2.2,
        1.6,
        {
          tone: "mid",
          steps: 1,
          bevel: 0,
          crease: "none",
        },
      ),
      "notifying.flag.pivot",
    ),
    owned(
      box(
        P,
        "notifying.flag",
        { x, y: BOX_Y + 15.4, w: 0.7, d: 4.6, r: 0.3 },
        SHELF + WALL - 4.8,
        4.2,
        {
          tone: "hi",
          steps: 1,
          bevel: 0,
          crease: "lo",
        },
      ),
      "notifying.flag.arm",
    ),
    disc(P, "notifying.flag.pivot", pivot, 0, 1.6, 1.2, { tone: "lo" }),
  ];
}

export function mailbox(P: Projection): Item {
  const shell = line(
    [3, LENGTH - 3]
      .map((s) => arcOf(frameAlong([0, BOX_Y, SHELF + WALL], [0, 1, 0]), s, HALF, P))
      .join(""),
    "faint",
  );
  return item(
    "notifying",
    [0, 0, 0],
    [
      halo(P, FOOT, 0, 2, 5),
      box(P, "notifying.foot", FOOT, 0, 2, { tone: "lo", bevel: 0.4 }),
      ...screws(P, corners(FOOT, 2), 2),
      box(P, "notifying.post", POST, 2, POST_TOP - 2, {
        bevel: 0.3,
        inner: line(sideSeam(POST, 12, P, 2), "faint").el,
      }),
      box(
        P,
        "notifying.shelf",
        { x: -4, y: BOX_Y + 3, w: 8, d: LENGTH - 6, r: 0.6 },
        POST_TOP,
        1.4,
        {
          tone: "lo",
          steps: 2,
          bevel: 0.2,
        },
      ),
      prism(P, "notifying.box", across, END, 0, LENGTH, { inner: shell.el }),
      door(P),
      dots([iso([0, FRONT, SHELF + 2.6], P)], 0.7, "hi"),
      ...flag(P),
    ],
  );
}
