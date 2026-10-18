import {
  type Plan,
  type Projection,
  corners,
  insetPlan,
  iso,
  knurl,
  planOutline,
} from "@anatomy/iso-kit";
import { arcOf, frameAlong } from "@anatomy/lathe";

import { type Drawn, type Item, box, disc, dots, halo, item, line, mark, post } from "./part";
import { faceY, roundRect } from "./face";

// Subtitles' machine: a typewriter with a sheet in it. It stands at rest
// until a Subtitles task reports its runs.

const BASE: Plan = { x: -18, y: -12, w: 36, d: 25, r: 2.4 };
const BASE_Z = 2;
const BASE_TOP = 8;
const HOOD: Plan = { x: -16.5, y: -12, w: 33, d: 11, r: 1.6 };
const HOOD_TOP = 14;
const PLATEN = { y: -8.2, z: HOOD_TOP + 4.3, r: 3, from: -14.5, to: 14.5 } as const;
const PAPER: Plan = { x: -8.5, y: HOOD.y, w: 17, d: 0.5, r: 0.1 };
const PAPER_Z = HOOD_TOP;
const PAPER_TOP = PAPER_Z + 21;
const KEY_ROWS = [
  { y: 1.6, z: BASE_TOP + 1.6, count: 8, from: -12.6 },
  { y: 5, z: BASE_TOP + 1.1, count: 8, from: -11.6 },
  { y: 8.4, z: BASE_TOP + 0.6, count: 7, from: -10.6 },
] as const;
const KEY_PITCH = 3.4;

function platen(P: Projection): Drawn[] {
  const frame = frameAlong([PLATEN.from, PLATEN.y, PLATEN.z], [1, 0, 0]);
  const length = PLATEN.to - PLATEN.from;
  return [
    box(P, "subtitles.carriage", { x: -15.5, y: -11.3, w: 31, d: 5, r: 0.8 }, HOOD_TOP, 1.2, {
      tone: "lo",
      steps: 2,
      bevel: 0,
    }),
    disc(P, "subtitles.knob.left", frame, -3, -0.4, 2.2, { tone: "mid" }),
    disc(P, "subtitles.platen", frame, -0.4, length + 0.4, PLATEN.r, {
      tone: "hi",
      crease: "lo",
      inner: line([4, length - 4].map((s) => arcOf(frame, s, PLATEN.r, P)).join(""), "faint").el,
    }),
    disc(P, "subtitles.knob.right", frame, length + 0.4, length + 3, 2.2, { tone: "mid" }),
  ];
}

function paper(P: Projection): Drawn[] {
  const face = PAPER.y + PAPER.d;
  const lines = [13, 11, 12.4, 7.6];
  return [
    box(P, "subtitles.paper", PAPER, PAPER_Z, PAPER_TOP - PAPER_Z, {
      tone: "hi",
      crease: "none",
      bevel: 0,
      steps: 1,
    }),
    mark(
      <g transform={faceY(P, face)}>
        {lines.map((length, index) => (
          <path
            key={length}
            className="iso-line"
            data-tone="lo"
            data-free=""
            d={`M${String(PAPER.x + 2)} ${String(-(PAPER_TOP - 3 - index * 2.4))}h${String(length)}`}
          />
        ))}
      </g>,
    ),
  ];
}

function keys(P: Projection): Drawn[] {
  return KEY_ROWS.flatMap((row, rowIndex) =>
    Array.from({ length: row.count }, (_, index) =>
      post(
        P,
        `subtitles.key.${String(rowIndex)}.${String(index)}`,
        [row.from + index * KEY_PITCH, row.y],
        1.2,
        row.z,
        1.2,
        { tone: "hi", steps: 12, crease: "none", bevel: 0.2 },
      ),
    ),
  );
}

function deck(P: Projection): Drawn[] {
  return KEY_ROWS.map((row, index) =>
    box(
      P,
      `subtitles.deck.${String(index)}`,
      { x: -15, y: row.y - 1.7, w: 30, d: 3.4, r: 0.6 },
      BASE_TOP,
      row.z - BASE_TOP,
      { tone: "lo", steps: 2, bevel: 0 },
    ),
  );
}

export function typewriter(P: Projection): Item {
  return item(
    "subtitles",
    [0, 0, 0],
    [
      halo(P, BASE, 0, 3, 6),
      ...corners(BASE, 3.4)
        .toSorted(([ax, ay], [bx, by]) => ax + ay - (bx + by))
        .map((at, index) =>
          post(P, `subtitles.foot.${String(index)}`, at, 1.8, 0, BASE_Z, { tone: "lo", steps: 16 }),
        ),
      box(P, "subtitles.base", BASE, BASE_Z, BASE_TOP - BASE_Z, { bevel: 0.7 }),
      mark(
        <g transform={faceY(P, BASE.y + BASE.d)}>
          <path
            className="iso-line"
            data-tone="faint"
            d={roundRect(BASE.x + 2, BASE_Z + 1.4, BASE.x + BASE.w - 2, BASE_TOP - 1.4, 0.8)}
          />
        </g>,
      ),
      box(P, "subtitles.hood", HOOD, BASE_TOP, HOOD_TOP - BASE_TOP, { bevel: 0.6 }),
      line(planOutline(insetPlan(HOOD, 1.4), HOOD_TOP, P, 3), "faint"),
      ...[-11, 11].map((x) =>
        post(P, `subtitles.spool.${String(x)}`, [x, -3.6], 2.4, HOOD_TOP, 1.2, {
          tone: "lo",
          steps: 20,
        }),
      ),
      line(
        [-11, 11].map((x) => knurl(x, -3.6, 2.4, HOOD_TOP, HOOD_TOP + 1.2, 16, P)).join(""),
        "faint",
      ),
      ...paper(P),
      ...platen(P),
      ...deck(P),
      ...keys(P),
      box(P, "subtitles.space", { x: -8, y: 10.6, w: 16, d: 1.8, r: 0.6 }, BASE_TOP, 1, {
        tone: "hi",
        steps: 2,
        crease: "none",
        bevel: 0.2,
      }),
      dots([iso([BASE.x + BASE.w - 2.6, BASE.y + BASE.d, BASE_Z + 3], P)], 0.38),
    ],
  );
}
