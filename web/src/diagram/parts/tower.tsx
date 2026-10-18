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
import { disc as discOf, fillet, frameAlong, ribsOf } from "@anatomy/lathe";
import { Solid } from "@anatomy/react/draw";

import {
  type Drawn,
  type Item,
  box,
  dots,
  faded,
  halo,
  item,
  line,
  mark,
  post,
  screws,
} from "./part";
import { faceX, faceY, roundRect } from "./face";
import { tube } from "./tube";

// Relaying's machine: a switchboard tower with two sockets plugged in. It
// stands still until S52 (Relay Node) gives it live figures.

const TOWER: Plan = { x: 0, y: 0, w: 58, d: 44, r: 3 };
const FRONT = TOWER.y + TOWER.d;
const TOP = 96;
const SOCKETS = [
  { x: 14, z: 44 },
  { x: 44, z: 44 },
] as const;

function plug(P: Projection, index: number, x: number, z: number): Drawn {
  const frame = frameAlong([x, FRONT, z], [0, 1, 0]);
  return {
    el: (
      <>
        <Solid paths={discOf(0, 5, 4.4, frame, P)} tone="mid" crease="lo" />
        {faded(ribsOf(frame, 0.4, 4.6, 4.4, 14, P, { fade: [0.08, 0.5] }), "lo").el}
        <Solid paths={discOf(5, 8, 2, frame, P)} tone="mid" crease="lo" />
      </>
    ),
    shapes: [
      { kind: "disc", name: `relaying.plug.${String(index)}`, frame, s0: 0, s1: 5, r: 4.4 },
      { kind: "disc", name: `relaying.plug.${String(index)}.tail`, frame, s0: 5, s1: 8, r: 2 },
    ],
  };
}

function lock(P: Projection): {
  readonly parts: Drawn[];
  readonly pipe: ReturnType<typeof tube>["pipe"];
} {
  const lx = TOWER.x + TOWER.w;
  const body: Plan = { x: lx, y: TOWER.y + 18, w: 4, d: 14, r: 1.4 };
  const route: Point3[] = [
    [lx + 2.2, body.y + 3.4, 66],
    [lx + 2.2, body.y + 3.4, 71],
    [lx + 2.2, body.y + body.d - 3.4, 71],
    [lx + 2.2, body.y + body.d - 3.4, 66],
  ];
  const shackle = tube(P, "relaying.lock.shackle", fillet(route, 3.2, 8), 1.1, { tone: "hi" });
  return {
    parts: [
      shackle,
      box(P, "relaying.lock", body, 52, 14, { tone: "hi", crease: "lo", bevel: 0.5 }),
      mark(
        <g transform={faceX(P, lx + 4)}>
          <circle className="keyhole" cx={-(body.y + 7)} cy={-59.5} r={1.3} />
          <path className="keyhole" d={`M${String(-(body.y + 7) - 0.5)} -59.2h1l0.4 3h-1.8z`} />
        </g>,
      ),
      dots([iso([lx + 1.4, body.y + 4, 56], P), iso([lx + 1.4, body.y + 10, 56], P)], 0.36),
    ],
    pipe: shackle.pipe,
  };
}

function vents(P: Projection): Drawn {
  const paths: string[] = [];
  for (let index = 0; index < 6; index++) {
    paths.push(
      pathOf(
        roundedPlan({ x: TOWER.y + 7 + index * 4.6, y: 78, w: 2, d: 12, r: 1 }, 3).map(([u, z]) =>
          iso([TOWER.x + TOWER.w, u, z], P),
        ),
        true,
      ),
    );
  }
  return line(paths.join(""), "lo");
}

function Meter() {
  return (
    <>
      <path
        className="iso-line"
        data-tone="lo"
        d={roundRect(TOWER.x + 6, 62, TOWER.x + 30, 88, 1.2)}
      />
      {Array.from({ length: 8 }, (_, index) => (
        <rect
          key={index}
          className="cpu-cell"
          x={TOWER.x + 8 + index * 2.7}
          y={-86}
          width={1.9}
          height={22}
        />
      ))}
      <path
        className="iso-line"
        data-tone="lo"
        data-free=""
        d={`M${String(TOWER.x + 34)} -80H${String(TOWER.x + 52)}M${String(TOWER.x + 34)} -77H${String(TOWER.x + 46)}`}
      />
    </>
  );
}

function Ports() {
  return (
    <>
      <path
        className="iso-line"
        data-tone="lo"
        d={roundRect(TOWER.x + 6, 14, TOWER.x + TOWER.w - 6, 28, 1)}
      />
      {Array.from({ length: 4 }, (_, index) => (
        <g key={index}>
          <circle className="port-led" cx={TOWER.x + 12 + index * 11} cy={-24} r={0.9} />
          <path
            className="iso-line"
            data-tone="mid"
            d={roundRect(TOWER.x + 8 + index * 11, 15.6, TOWER.x + 16 + index * 11, 21, 0.6)}
          />
        </g>
      ))}
    </>
  );
}

function panel(P: Projection): Drawn {
  return mark(
    <g transform={faceY(P, FRONT)}>
      <path
        className="iso-line"
        data-tone="faint"
        d={roundRect(TOWER.x + 3, 8, TOWER.x + TOWER.w - 3, TOP - 3, 1.5)}
      />
      <Meter />
      {SOCKETS.map((socket) => (
        <g key={socket.x}>
          <path
            className="iso-line"
            data-tone="lo"
            d={roundRect(socket.x - 9, socket.z - 9, socket.x + 9, socket.z + 9, 1.6)}
          />
          <circle className="socket" cx={socket.x} cy={-socket.z} r={6} />
        </g>
      ))}
      <Ports />
    </g>,
  );
}

const LAMPS_AT = [TOWER.x + 12, TOWER.x + 22, TOWER.x + 32];

function lamps(P: Projection): Drawn[] {
  return LAMPS_AT.flatMap((x, index) => [
    post(P, `relaying.lamp.${String(index)}.base`, [x, FRONT - 6], 3, TOP + 3, 1.4, {
      tone: "lo",
      steps: 20,
    }),
    post(P, `relaying.lamp.${String(index)}`, [x, FRONT - 6], 2.2, TOP + 4.4, 2.6, { steps: 20 }),
  ]);
}

export function tower(P: Projection): Item {
  const latch = lock(P);
  return item(
    "relaying",
    [TOWER.x + TOWER.w / 2, TOWER.y + TOWER.d / 2, 0],
    [
      halo(P, TOWER, 0, 4, 8),
      box(P, "relaying.plinth", insetPlan(TOWER, 2), 0, 5, { tone: "lo", bevel: 0 }),
      box(P, "relaying.body", TOWER, 5, TOP - 5, {
        bevel: 0.8,
        inner: line([24, 48, 72].map((z) => sideSeam(TOWER, z, P, 3)).join(""), "faint").el,
      }),
      box(P, "relaying.lid", insetPlan(TOWER, -1), TOP, 3, { bevel: 0.8 }),
      ...screws(P, corners(TOWER, 3), TOP + 3),
      vents(P),
      panel(P),
      dots(
        SOCKETS.flatMap((socket) => [
          iso([socket.x - 7, FRONT, socket.z - 7], P),
          iso([socket.x + 7, FRONT, socket.z + 7], P),
        ]),
        0.4,
      ),
      ...SOCKETS.map((socket, index) => plug(P, index, socket.x, socket.z)),
      ...lamps(P),
      ...latch.parts,
    ],
    [latch.pipe],
  );
}
