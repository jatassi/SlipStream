import { type Plan, type Plane, type Projection, iso, sideArc } from "@anatomy/iso-kit";
import { dotsOf, frameAlong } from "@anatomy/lathe";

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
} from "./part";
import { faceY, roundRect } from "./face";

// Downloading's antenna: a mast with a Yagi head that takes pieces in, and a
// receiver halfway up that hands them to the chute.

const ANT: Plane = [-150, -60];
const ANT_TOP = 146;
const BOOM = { from: -178, to: -142 };
export const RX = { x: -158, y: -68, w: 16, d: 14, r: 1.5, z: 100, h: 16 };

function receiver(P: Projection): Drawn[] {
  return [
    box(P, "downloading.antenna.receiver", RX, RX.z, RX.h, { bevel: 0.6 }),
    mark(
      <g transform={faceY(P, RX.y + RX.d)}>
        <path
          className="iso-line"
          data-tone="faint"
          d={roundRect(RX.x + 2.4, RX.z + 3, RX.x + RX.w - 2.4, RX.z + RX.h - 3, 1.2)}
        />
        <path
          className="iso-line"
          data-tone="lo"
          d={roundRect(RX.x + 3.4, RX.z + 9.5, RX.x + 9.5, RX.z + 11.8, 1)}
        />
      </g>,
    ),
    group({ "data-led": "rx" }, [
      dots([iso([RX.x + RX.w - 4, RX.y + RX.d, RX.z + 10.6], P)], 0.85, "hi"),
    ]),
    dots(
      [
        iso([RX.x + 2.2, RX.y + RX.d, RX.z + 1.8], P),
        iso([RX.x + RX.w - 2.2, RX.y + RX.d, RX.z + 1.8], P),
      ],
      0.4,
    ),
  ];
}

const ELEMENTS = [-174, -167, -160, -153, -146].map((x, index) => ({ x, half: 7 + index * 1.5 }));

function element(
  P: Projection,
  side: "back" | "front",
  { x, half }: { x: number; half: number },
  index: number,
): Drawn {
  const y = ANT[1];
  return box(
    P,
    `downloading.antenna.${side}.${String(index)}`,
    side === "back"
      ? { x: x - 0.6, y: y - half, w: 1.2, d: half - 1.2, r: 0.5 }
      : { x: x - 0.6, y: y + 1.2, w: 1.2, d: half - 1.2, r: 0.5 },
    ANT_TOP + 0.2,
    1.2,
    { steps: 2, bevel: 0, crease: "none" },
  );
}

function head(P: Projection): Drawn[] {
  const y = ANT[1];
  const boom: Plan = { x: BOOM.from, y: y - 1.2, w: BOOM.to - BOOM.from, d: 2.4, r: 0.6 };
  return [
    post(P, "downloading.antenna.head", ANT, 3.4, ANT_TOP - 3, 3, { steps: 18, bevel: 0.3 }),
    ...ELEMENTS.map((e, index) => element(P, "back", e, index)),
    box(P, "downloading.antenna.boom", boom, ANT_TOP, 2.4, { steps: 2, bevel: 0 }),
    ...ELEMENTS.map((e, index) => element(P, "front", e, index)),
    dots(
      ELEMENTS.map((e) => iso([e.x, y, ANT_TOP + 2.4], P)),
      0.4,
    ),
    post(P, "downloading.antenna.whip", ANT, 1, ANT_TOP + 2.4, 10, { steps: 12, crease: "none" }),
    group({ "data-led": "beacon" }, [
      post(P, "downloading.antenna.beacon", ANT, 2, ANT_TOP + 12.4, 3, {
        tone: "hi",
        steps: 16,
        lit: true,
      }),
    ]),
  ];
}

export function antenna(P: Projection): Item {
  const [x, y] = ANT;
  return item(
    "downloading.antenna",
    [x, y, 0],
    [
      halo(P, { x: x - 12, y: y - 12, w: 24, d: 24, r: 12 }, 0, 1, 5),
      post(P, "downloading.antenna.foot", ANT, 11, 0, 2, { tone: "lo", steps: 40, bevel: 0.5 }),
      dots(dotsOf(frameAlong([x, y, 0], [0, 0, 1]), 2, 8.4, 6, P, { all: true }), 0.45),
      post(P, "downloading.antenna.collar", ANT, 5, 2, 5, { steps: 28, bevel: 0.4 }),
      post(P, "downloading.antenna.mast", ANT, 2.6, 7, RX.z - 7, {
        steps: 20,
        crease: "none",
        inner: line([30, 60].map((z) => sideArc(x, y, 2.6, z, P, 12)).join(""), "faint").el,
      }),
      ...[40, 70].map((z) =>
        owned(
          post(P, `downloading.antenna.band.${String(z)}`, ANT, 3.6, z - 1.5, 3, {
            steps: 20,
            bevel: 0.3,
          }),
          "downloading.antenna.mast",
        ),
      ),
      ...receiver(P),
      post(P, "downloading.antenna.upper", ANT, 2.2, RX.z + RX.h, ANT_TOP - RX.z - RX.h - 3, {
        steps: 18,
        crease: "none",
      }),
      ...head(P),
    ],
  );
}
