import {
  type Projection,
  corners,
  insetPlan,
  iso,
  knurl,
  pathOf,
  roundedPlan,
  sideSeam,
} from "@anatomy/iso-kit";
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
  post,
  screws,
} from "./part";
import { faceY, roundRect } from "./face";
import { COLLAR_TOP, HOUSING, HOUSING_TOP, LID_TOP, TURNTABLE_TOP, radarHead } from "./radar-head";

// Searching's machine: a radar dish on a turntable. The live loop turns the
// head about the turntable's axis while a search runs.

function housingFace(P: Projection): Drawn {
  const y = HOUSING.y + HOUSING.d;
  return mark(
    <g transform={faceY(P, y)}>
      <path className="iso-line" data-tone="faint" d={roundRect(-7.4, 2.6, 7.4, 16, 1)} />
      <circle className="window" cx={-2.6} cy={-10.6} r={3.4} />
      <path className="iso-line" data-tone="lo" d="M-2.6 -14v6.8M-6 -10.6h6.8" />
      <path className="iso-line" data-tone="lo" d={roundRect(-6.2, 4, 1, 5.6, 0.6)} />
    </g>,
  );
}

function vents(P: Projection): Drawn {
  const x = HOUSING.x + HOUSING.w;
  const slots: string[] = [];
  for (let index = 0; index < 4; index++) {
    slots.push(
      pathOf(
        roundedPlan({ x: HOUSING.y + 4 + index * 3, y: 9, w: 1.4, d: 6, r: 0.7 }, 3).map(([u, z]) =>
          iso([x, u, z], P),
        ),
        true,
      ),
    );
  }
  return line(slots.join(""), "lo");
}

export function dish(P: Projection): Item {
  const lid = insetPlan(HOUSING, -0.8);
  return item(
    "searching",
    [0, 0, 0],
    [
      halo(P, { x: -15, y: -15, w: 30, d: 30, r: 15 }, 0, 1, 5),
      post(P, "searching.foot", [0, 0], 14, 0, 2, { tone: "lo", steps: 40, bevel: 0.5 }),
      dots(dotsOf(frameAlong([0, 0, 0], [0, 0, 1]), 2, 12, 8, P, { all: true }), 0.45),
      box(P, "searching.housing", HOUSING, 2, HOUSING_TOP - 2, {
        bevel: 0.6,
        inner: line(sideSeam(HOUSING, 12, P, 3), "faint").el,
      }),
      housingFace(P),
      group({ "data-led": "search", className: "led" }, [
        dots([iso([5, HOUSING.y + HOUSING.d, 11.4], P)], 0.85, "hi"),
      ]),
      vents(P),
      box(P, "searching.lid", lid, HOUSING_TOP, LID_TOP - HOUSING_TOP, { bevel: 0.5 }),
      ...screws(P, corners(lid, 2.2), LID_TOP),
      post(P, "searching.turntable", [0, 0], 6.4, LID_TOP, TURNTABLE_TOP - LID_TOP, {
        steps: 32,
        bevel: 0.3,
      }),
      line(knurl(0, 0, 6.4, LID_TOP, TURNTABLE_TOP, 30, P), "lo"),
      post(P, "searching.collar", [0, 0], 3, TURNTABLE_TOP, COLLAR_TOP - TURNTABLE_TOP, {
        steps: 24,
        bevel: 0.3,
      }),
      radarHead(P),
    ],
  );
}
