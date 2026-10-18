import { type Flat, type Plan, type Point3, type Projection, iso, pathOf } from "@anatomy/iso-kit";
import { type Frame, dot3, frameAlong, hullOf, viewOf } from "@anatomy/lathe";

import type { Drawn } from "./part";

// The radar dish's head, drawn from projected circles so the live loop can
// redraw it at any heading every frame.

export const HOUSING: Plan = { x: -9, y: -9, w: 18, d: 18, r: 1.5 };
export const HOUSING_TOP = 18;
export const LID_TOP = HOUSING_TOP + 1.6;
export const TURNTABLE_TOP = LID_TOP + 2.4;
export const COLLAR_TOP = TURNTABLE_TOP + 3;
const PIVOT: Point3 = [0, 0, COLLAR_TOP + 12];
const TILT = (24 * Math.PI) / 180;
const RIM = 13;
const FOCUS = 10.6;
const HUB = 3;
const SHELL = 0.8;

/** Where the dish faces at rest, in degrees about the turntable's axis. */
export const REST_HEADING = 117;

const bowlDepth = (r: number): number => (r * r) / (4 * FOCUS);
const DEPTH = bowlDepth(RIM);

/** The head's axis pointing at a heading, tilted up, from the pivot. */
export const headFrame = (heading: number): Frame => {
  const radians = (heading * Math.PI) / 180;
  return frameAlong(PIVOT, [
    Math.cos(radians) * Math.cos(TILT),
    Math.sin(radians) * Math.cos(TILT),
    Math.sin(TILT),
  ]);
};

/** How one of the head's paths is painted: a shade, the lit top, or a line in a tone. */
type Paint = "0" | "1" | "2" | "3" | "top" | "hi" | "mid" | "lo" | "faint";

export type HeadPath = { readonly paint: Paint; readonly d: string };

/** How many paths the head draws in every pose, so the live loop rewrites them in place. */
export const HEAD_PATHS = 14;

const RING = 24;

function circle(P: Projection, F: Frame, s: number, r: number): Flat[] {
  const out: Flat[] = [];
  for (let index = 0; index < RING; index++) {
    const t = (index / RING) * Math.PI * 2;
    const c = Math.cos(t) * r;
    const n = Math.sin(t) * r;
    out.push(
      iso(
        [
          F.o[0] + F.a[0] * s + c * F.u[0] + n * F.v[0],
          F.o[1] + F.a[1] * s + c * F.u[1] + n * F.v[1],
          F.o[2] + F.a[2] * s + c * F.u[2] + n * F.v[2],
        ],
        P,
      ),
    );
  }
  return out;
}

const closed = (points: readonly Flat[]): string => pathOf(points, true);
const round = (P: Projection, F: Frame, s0: number, s1: number, r: number): string =>
  closed(hullOf([...circle(P, F, s0, r), ...circle(P, F, s1, r)]));

/** A short round part on the axis: its body, then its end toward the viewer, then its outline. */
function plug(
  P: Projection,
  F: Frame,
  s0: number,
  s1: number,
  r: number,
  facing: boolean,
): HeadPath[] {
  const body = round(P, F, s0, s1, r);
  return [
    { paint: "1", d: body },
    { paint: "top", d: closed(circle(P, F, facing ? s1 : s0, r)) },
    { paint: "mid", d: body },
  ];
}

/**
 * The head at a heading, in painter's order, about the machine's own
 * origin: the neck, the hub, the bowl and its feed. Seen from the front the
 * bowl shows its dark hollow and the feed stands in it; from behind, its
 * rounded back hides the feed.
 */
export function headOf(P: Projection, heading: number): readonly HeadPath[] {
  const paths = headPaths(P, heading);
  return [
    ...paths,
    ...Array.from({ length: HEAD_PATHS - paths.length }, (): HeadPath => ({
      paint: "faint",
      d: "",
    })),
  ];
}

function headPaths(P: Projection, heading: number): HeadPath[] {
  const at: Projection = { ...P, origin: [0, 0] };
  const F = headFrame(heading);
  const facing = dot3(F.a, viewOf(at)) >= 0;
  const neckFrame = frameAlong([0, 0, COLLAR_TOP], [0, 0, 1]);
  const neck = round(at, neckFrame, 0, PIVOT[2] - COLLAR_TOP, 1.4);
  const neckPaths: HeadPath[] = [
    { paint: "2", d: neck },
    { paint: "mid", d: neck },
  ];
  const lip = HUB + DEPTH;
  const rim = round(at, F, lip, lip + SHELL, RIM);
  const feed = [
    {
      paint: "mid" as const,
      d: pathOf([iso(pointAt(F, HUB + SHELL), at), iso(pointAt(F, FOCUS + HUB), at)]),
    },
    ...plug(at, F, FOCUS + HUB, FOCUS + HUB + 1.8, 1.5, facing),
  ];
  if (facing) {
    return [
      ...neckPaths,
      { paint: "2", d: rim },
      { paint: "0", d: closed(circle(at, F, lip + SHELL, RIM - 0.6)) },
      { paint: "faint", d: closed(circle(at, F, HUB + SHELL + bowlDepth(6.5), 6.5)) },
      { paint: "hi", d: rim },
      { paint: "hi", d: closed(circle(at, F, HUB + SHELL, 1.1)) },
      ...feed,
    ];
  }
  const back = closed(hullOf([...circle(at, F, lip + SHELL, RIM), ...circle(at, F, HUB, 3)]));
  return [
    ...feed,
    { paint: "1", d: back },
    { paint: "3", d: closed(circle(at, F, HUB + bowlDepth(7.5), 7.5)) },
    { paint: "faint", d: closed(circle(at, F, lip, RIM)) },
    { paint: "hi", d: back },
    ...plug(at, F, -0.6, HUB + 0.2, 2.6, false),
    ...neckPaths,
  ];
}

const pointAt = (F: Frame, s: number): Point3 => [
  F.o[0] + F.a[0] * s,
  F.o[1] + F.a[1] * s,
  F.o[2] + F.a[2] * s,
];

/** The attributes that paint a head path. */
export function paintOf(paint: Paint): Readonly<Record<string, string>> {
  if (paint === "top") {
    return { class: "iso-top", "data-shade": "", "data-tone": "" };
  }
  if (paint === "0" || paint === "1" || paint === "2" || paint === "3") {
    return { class: "iso-shade", "data-shade": paint, "data-tone": "" };
  }
  return { class: "iso-line", "data-shade": "", "data-tone": paint };
}

function headShapes(): Drawn["shapes"] {
  const F = headFrame(REST_HEADING);
  return [
    {
      kind: "disc",
      name: "searching.neck",
      owner: "searching.hub",
      frame: frameAlong([0, 0, COLLAR_TOP], [0, 0, 1]),
      s0: 0,
      s1: PIVOT[2] - COLLAR_TOP,
      r: 1.4,
    },
    {
      kind: "disc",
      name: "searching.hub",
      owner: "searching.dish",
      frame: F,
      s0: -0.6,
      s1: HUB + 0.2,
      r: 2.6,
    },
    { kind: "disc", name: "searching.dish", frame: F, s0: HUB, s1: HUB + DEPTH + SHELL, r: RIM },
    {
      kind: "disc",
      name: "searching.feed",
      owner: "searching.dish",
      frame: F,
      s0: HUB + SHELL + 0.2,
      s1: FOCUS + HUB + 1.8,
      r: 1.5,
    },
  ];
}

/** The dish's head at rest: the neck, the hub, the bowl and its feed. */
export function radarHead(P: Projection): Drawn {
  return {
    el: (
      <g
        data-dish=""
        className="dish"
        transform={`translate(${String(P.origin[0])} ${String(P.origin[1])})`}
      >
        {headOf(P, REST_HEADING).map(({ paint, d }, index) => {
          const { class: className, "data-shade": shade, "data-tone": tone } = paintOf(paint);
          return (
            // oxlint-disable-next-line react/no-array-index-key -- the live loop rewrites each path in place
            <path key={index} className={className} data-shade={shade} data-tone={tone} d={d} />
          );
        })}
      </g>
    ),
    shapes: headShapes(),
  };
}
