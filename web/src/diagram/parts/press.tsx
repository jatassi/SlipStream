import {
  type Plan,
  type Projection,
  corners,
  insetPlan,
  knurl,
  lineOnTop,
  pathOf,
  planOutline,
} from "@anatomy/iso-kit";
import { arcOf, disc as discOf, frameAlong, ribsOf } from "@anatomy/lathe";

import { Solid } from "@anatomy/react/draw";

import { type Drawn, type Item, box, faded, halo, item, line, mark, post, screws } from "./part";
import { faceX, faceY, roundRect } from "./face";

// Converting's machine: a roller press that film runs through. It stands
// still until S44 (Conversion) gives it live figures.

const PRESS: Plan = { x: -8, y: 26, w: 150, d: 68, r: 4 };
const NIP_X = 76;
const ROLL_R = 7;
const ROLL_LOW_Z = 22;
const ROLL_HIGH_Z = ROLL_LOW_Z + ROLL_R * 2 + 0.8;
const ROLL_Y = [36, 84] as const;
const IN_X0 = -2;
const OUT_X1 = 128;
const RIB_COUNT = 24;
const FRAME_TOP = 47;
const LANES = 4;

function roller(P: Projection, name: string, z: number): Drawn {
  const frame = frameAlong([NIP_X, ROLL_Y[0], z], [0, 1, 0]);
  const length = ROLL_Y[1] - ROLL_Y[0];
  const seams = line([8, 40].map((s) => arcOf(frame, s, ROLL_R, P)).join(""), "faint");
  return {
    el: (
      <>
        <Solid paths={discOf(0, length, ROLL_R, frame, P)} tone="hi" crease="lo" inner={seams.el} />
        {
          faded(ribsOf(frame, 0.4, length - 0.4, ROLL_R, RIB_COUNT, P, { fade: [0.08, 0.5] }), "lo")
            .el
        }
      </>
    ),
    shapes: [{ kind: "disc", name, frame, s0: 0, s1: length, r: ROLL_R }],
  };
}

function gearPath(cx: number, cz: number, r: number, teeth: number): string {
  const points: [number, number][] = [];
  for (let index = 0; index < teeth * 4; index++) {
    const angle = (index / (teeth * 4)) * Math.PI * 2;
    const radius = index % 4 < 2 ? r + 1.3 : r;
    points.push([cx + radius * Math.cos(angle), -(cz + radius * Math.sin(angle))]);
  }
  return pathOf(points, true);
}

function stand(P: Projection, name: string, y: number): Drawn[] {
  const knob: [number, number] = [NIP_X, y + 3.5];
  return [
    box(P, name, { x: NIP_X - 13, y, w: 26, d: 7, r: 1.4 }, 10, FRAME_TOP - 10, { bevel: 0.6 }),
    ...screws(
      P,
      [
        [NIP_X - 9, y + 3.5],
        [NIP_X + 9, y + 3.5],
      ],
      FRAME_TOP,
    ),
    post(P, `${name}.screw`, knob, 1.6, FRAME_TOP, 3, { steps: 14 }),
    post(P, `${name}.knob`, knob, 3.4, FRAME_TOP + 3, 3, { steps: 24, bevel: 0.3 }),
    line(knurl(knob[0], knob[1], 3.4, FRAME_TOP + 3, FRAME_TOP + 6, 22, P), "lo"),
  ];
}

function bedFace(P: Projection): Drawn {
  return mark(
    <g transform={faceY(P, PRESS.y + PRESS.d)}>
      <path
        className="iso-line"
        data-tone="lo"
        d={roundRect(PRESS.x + 6, 4.6, PRESS.x + 34, 8.6, 1)}
      />
      <path className="window" d={roundRect(PRESS.x + 7, 5.2, PRESS.x + 22, 8, 0.6)} />
      {Array.from({ length: LANES }, (_, index) => (
        <circle key={index} className="lamp-dot" cx={PRESS.x + 44 + index * 5} cy={-6.6} r={1.1} />
      ))}
      <path
        className="iso-line"
        data-tone="lo"
        d={roundRect(PRESS.x + 40.6, 4.6, PRESS.x + 62.4, 8.6, 1)}
      />
      <circle className="window" cx={PRESS.x + 104} cy={-6.6} r={2.4} />
      <path
        className="iso-line"
        data-tone="hi"
        data-free="end"
        d={`M${String(PRESS.x + 104)} -6.6l-1.7 1.2`}
      />
      <circle className="hub" cx={PRESS.x + 104} cy={-6.6} r={0.5} />
    </g>,
  );
}

function gears(P: Projection): Drawn {
  return mark(
    <g transform={faceY(P, ROLL_Y[1] + 7)}>
      <path className="gear-back" d={gearPath(NIP_X, ROLL_LOW_Z, 6.4, 14)} />
      {[ROLL_LOW_Z, ROLL_HIGH_Z].map((z) => (
        <g key={z}>
          <path className="gear" d={gearPath(NIP_X, z, 6.4, 14)} />
          <path
            className="iso-line"
            data-tone="mid"
            d={roundRect(NIP_X - 4.4, z - 0.5, NIP_X + 4.4, z + 0.5, 0.5, 2)}
          />
          <circle className="hub" cx={NIP_X} cy={-z} r={1.6} />
        </g>
      ))}
    </g>,
  );
}

const OUTBOX: Plan = { x: 128, y: 44, w: 12, d: 34, r: 1.5 };

function outbox(P: Projection): Drawn[] {
  return [
    box(P, "converting.outbox", OUTBOX, 10, 26, { bevel: 0.6 }),
    mark(
      <g transform={faceX(P, OUTBOX.x + OUTBOX.w)}>
        <path
          className="iso-line"
          data-tone="faint"
          d={roundRect(-(OUTBOX.y + OUTBOX.d - 3), 13, -(OUTBOX.y + 3), 33, 1)}
        />
      </g>,
    ),
    ...screws(
      P,
      [
        [OUTBOX.x + 3, OUTBOX.y + 3],
        [OUTBOX.x + 3, OUTBOX.y + OUTBOX.d - 3],
      ],
      36,
    ),
  ];
}

function tables(P: Projection): Drawn[] {
  const grooves: string[] = [];
  for (let index = 0; index <= 4; index++) {
    grooves.push(lineOnTop([IN_X0, 38 + index * 10], [NIP_X - 8, 38 + index * 10], 28.2, P));
  }
  const leg = { tone: "lo", steps: 2, bevel: 0.3 } as const;
  return [
    ...[2, 40].map((x) =>
      box(P, `converting.leg.${String(x)}`, { x, y: 38, w: 6, d: 46, r: 1 }, 10, 15.6, leg),
    ),
    box(
      P,
      "converting.table.in",
      { x: IN_X0, y: 37, w: NIP_X - 8 - IN_X0, d: 47, r: 1.2 },
      25.6,
      2.6,
      { bevel: 0.5 },
    ),
    line(grooves.join(""), "faint"),
    box(P, "converting.leg.out", { x: 116, y: 40, w: 6, d: 42, r: 1 }, 10, 15.6, leg),
    box(
      P,
      "converting.table.out",
      { x: NIP_X + 8, y: 39, w: OUT_X1 - NIP_X - 8, d: 44, r: 1.2 },
      25.6,
      2.6,
      { bevel: 0.5 },
    ),
  ];
}

export function press(P: Projection): Item {
  return item(
    "converting",
    [PRESS.x + PRESS.w / 2, PRESS.y + PRESS.d / 2, 0],
    [
      halo(P, PRESS, 0, 4, 7),
      ...corners(PRESS, 6)
        .toSorted(([ax, ay], [bx, by]) => ax + ay - (bx + by))
        .map((at, index) =>
          post(P, `converting.foot.${String(index)}`, at, 3.4, 0, 3, { tone: "lo", steps: 18 }),
        ),
      box(P, "converting.bed", PRESS, 3, 7, { steps: 4, bevel: 0.8 }),
      line(planOutline(insetPlan(PRESS, 3), 10, P, 4), "faint"),
      ...screws(P, corners(PRESS, 6), 10),
      bedFace(P),
      ...stand(P, "converting.stand.back", ROLL_Y[0] - 7),
      ...tables(P),
      roller(P, "converting.roller.low", ROLL_LOW_Z),
      roller(P, "converting.roller.high", ROLL_HIGH_Z),
      ...stand(P, "converting.stand.front", ROLL_Y[1]),
      gears(P),
      ...outbox(P),
    ],
  );
}
