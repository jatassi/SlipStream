import { type Plan, type Plane, type Projection, corners, iso, sideSeam } from "@anatomy/iso-kit";
import { frameOf } from "@anatomy/lathe";
import { FaceText } from "@anatomy/react/draw";

import {
  type Drawn,
  type Item,
  box,
  dots,
  halo,
  item,
  line,
  mark,
  owned,
  prism,
  screws,
} from "./part";
import { faceX, faceY, roundRect } from "./face";

// Serving's machine: a ticket booth whose "Open" sign is lit while its Node
// is online and listening for browsers.

const PLINTH: Plan = { x: -16, y: -13, w: 32, d: 26, r: 1.6 };
const BODY: Plan = { x: -14.5, y: -11.5, w: 29, d: 22, r: 1.2 };
const BODY_Z = 2.6;
const EAVE = BODY_Z + 25;
const FRONT = BODY.y + BODY.d;
const WINDOW = { u0: -10, u1: 10, z0: BODY_Z + 10.4, z1: BODY_Z + 21 } as const;
const SIGN = { u0: -8, u1: 8, z0: EAVE + 8.4, z1: EAVE + 13.4 } as const;

/** The roof's gable: eaves over the body's front and back, a ridge across. */
const GABLE: Plane[] = [
  [BODY.y - 2.4, 0],
  [FRONT + 2.4, 0],
  [FRONT + 2.4, 1],
  [(BODY.y + FRONT) / 2, 7.4],
  [BODY.y - 2.4, 1],
];

function front(P: Projection): Drawn {
  return mark(
    <g transform={faceY(P, FRONT)}>
      <path
        className="iso-line"
        data-tone="lo"
        d={roundRect(WINDOW.u0 - 1.2, WINDOW.z0 - 1.2, WINDOW.u1 + 1.2, WINDOW.z1 + 1.2, 1)}
      />
      <path className="window" d={roundRect(WINDOW.u0, WINDOW.z0, WINDOW.u1, WINDOW.z1, 0.6)} />
      <path
        className="iso-line"
        data-tone="lo"
        d={`M${String(WINDOW.u0)} ${String(-(WINDOW.z1 - 3))}H${String(WINDOW.u1)}`}
      />
      <path
        className="iso-line"
        data-tone="lo"
        d={roundRect(-4, WINDOW.z0, 4, WINDOW.z0 + 1.6, 0.8)}
      />
      <circle className="hub" cx={0} cy={-(WINDOW.z0 + 3)} r={1.4} />
      <path
        className="iso-line"
        data-tone="faint"
        d={roundRect(-11.6, BODY_Z + 1.6, 11.6, BODY_Z + 7, 0.8)}
      />
      <path
        className="iso-line"
        data-tone="lo"
        d={roundRect(-4, BODY_Z + 3.6, 4, BODY_Z + 4.8, 0.5)}
      />
    </g>,
  );
}

function sign(P: Projection, lit: boolean): Drawn[] {
  const plan: Plan = { x: SIGN.u0, y: -1.2, w: SIGN.u1 - SIGN.u0, d: 1.2, r: 0.3 };
  return [
    ...[SIGN.u0 + 2, SIGN.u1 - 2].map((x) =>
      owned(
        box(
          P,
          `serving.sign.post.${String(x)}`,
          { x: x - 0.4, y: -1, w: 0.8, d: 0.8, r: 0.2 },
          EAVE + 6.4,
          SIGN.z0 - EAVE - 6.4,
          {
            tone: "lo",
            steps: 1,
            bevel: 0,
            crease: "none",
          },
        ),
        "serving.roof",
      ),
    ),
    box(P, "serving.sign", plan, SIGN.z0, SIGN.z1 - SIGN.z0, { tone: "mid", bevel: 0.3 }),
    mark(
      <g transform={faceY(P, 0)} data-open-sign={lit ? "lit" : ""}>
        <path
          className="open-glow"
          d={roundRect(SIGN.u0 + 0.8, SIGN.z0 + 0.8, SIGN.u1 - 0.8, SIGN.z1 - 0.8, 0.6)}
        />
        <path
          className="iso-line"
          data-tone="lo"
          d={roundRect(SIGN.u0 + 0.8, SIGN.z0 + 0.8, SIGN.u1 - 0.8, SIGN.z1 - 0.8, 0.6)}
        />
        <FaceText
          transform={`translate(0 ${String(-(SIGN.z0 + 1.7))})`}
          size={3}
          tone="hi"
          anchor="middle"
        >
          OPEN
        </FaceText>
      </g>,
    ),
  ];
}

function sideDoor(P: Projection): Drawn {
  const x = BODY.x + BODY.w;
  return mark(
    <g transform={faceX(P, x)}>
      <path className="iso-line" data-tone="lo" d={roundRect(-4, BODY_Z + 1.4, 4, EAVE - 4, 0.8)} />
      <path
        className="iso-line"
        data-tone="faint"
        d={roundRect(-2.6, EAVE - 11, 2.6, EAVE - 5.4, 0.5)}
      />
      <circle className="hub" cx={2.6} cy={-(BODY_Z + 10)} r={0.7} />
    </g>,
  );
}

/** Serving's ticket booth; its sign is lit while its Node listens for browsers. */
export function booth(P: Projection, lit: boolean): Item {
  const roof = frameOf([BODY.x - 2, 0, EAVE], [1, 0, 0], [0, 1, 0], [0, 0, 1]);
  return item(
    "serving",
    [0, 0, 0],
    [
      halo(P, PLINTH, 0, 3, 6),
      box(P, "serving.plinth", PLINTH, 0, BODY_Z, { tone: "lo", bevel: 0.4 }),
      ...screws(P, corners(PLINTH, 2), BODY_Z),
      box(P, "serving.body", BODY, BODY_Z, EAVE - BODY_Z, {
        bevel: 0.6,
        inner: line(sideSeam(BODY, BODY_Z + 8, P, 3), "faint").el,
      }),
      front(P),
      box(P, "serving.counter", { x: -12, y: FRONT, w: 24, d: 3.6, r: 0.6 }, WINDOW.z0 - 2.4, 1.2, {
        tone: "hi",
        crease: "lo",
        bevel: 0.3,
      }),
      dots(
        [iso([-11, FRONT + 1.8, WINDOW.z0 - 1.2], P), iso([11, FRONT + 1.8, WINDOW.z0 - 1.2], P)],
        0.36,
      ),
      sideDoor(P),
      prism(P, "serving.roof", roof, GABLE, 0, BODY.w + 4),
      ...sign(P, lit),
    ],
  );
}
