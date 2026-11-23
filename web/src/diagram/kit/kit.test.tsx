import "@anatomy/iso.css";

import * as IsoKit from "@anatomy/iso-kit";
import * as Lathe from "@anatomy/lathe";
import * as Draw from "@anatomy/react/draw";
import * as Live from "@anatomy/react/live";
import * as Tube from "@anatomy/tube";
import { useRef } from "react";
import { expect, test } from "vitest";
import { page } from "vitest/browser";

import { renderKit } from "../../test/render-kit";

const plan: IsoKit.Plan = { x: 0, y: 0, w: 60, d: 40, r: 4 };
const P = IsoKit.fitProjection(
  [
    [0, 0, 0],
    [60, 40, 12],
  ],
  240,
  160,
  { pad: 16 },
);
const slab = IsoKit.slabOf(plan, 0, 8, P);
const route = Lathe.fillet(
  [
    [0, 20, 8],
    [30, 20, 8],
    [30, 40, 8],
  ],
  4,
);
const tube = Tube.tubePieces(route, 1.2, P);

function Counter() {
  const { anchor, say } = Live.useReadout<SVGGElement>();
  const still = Live.useStillness();
  const frames = useRef(0);
  Live.useLoop(() => {
    frames.current += 1;
    say(`${String(frames.current)} frames${still ? ", still" : ""}`);
    return frames.current < 3;
  }, true);
  return <g ref={anchor} />;
}

function Figure() {
  return (
    <div className="iso">
      <Draw.Plate
        fig="Fig 1"
        title="One tile"
        readout="waiting"
        keys={[{ mark: "raised", label: "Tile" }]}
      >
        <Draw.IsoFigure width={240} height={160} label="A tile with a pipe">
          <Draw.Solid paths={slab} />
          <Draw.Line d={IsoKit.planOutline(IsoKit.insetPlan(plan, 6), 8, P)} tone="mid" />
          <Draw.Wire from={IsoKit.iso([0, 0, 8], P)} to={IsoKit.iso([60, 40, 8], P)} />
          <Draw.Dots points={[IsoKit.iso([30, 20, 8], P)]} />
          <Draw.FaceText transform={IsoKit.topMatrix([10, 10], 8, P)}>Controller</Draw.FaceText>
          {tube.map((piece) => (
            <path key={piece.from} className="tb-body" d={piece.body} />
          ))}
          <Counter />
        </Draw.IsoFigure>
      </Draw.Plate>
    </div>
  );
}

test("the kit's React helpers draw and animate a figure in the Vite app", async () => {
  const root = renderKit(<Figure />);
  const svg = page.getByRole("img", { name: "A tile with a pipe" });
  await expect.element(svg).toBeVisible();
  const fill = root.querySelector(".iso-fill");
  expect(fill?.getAttribute("d")).toBe(slab.fill);
  expect(root.querySelector(".iso-face-text")?.textContent).toBe("Controller");
  expect(tube.length).toBeGreaterThan(0);
  expect(root.querySelectorAll(".tb-body")).toHaveLength(tube.length);
  await expect
    .poll(() => root.querySelector<HTMLElement>(".iso-stage")?.dataset["inview"])
    .toBe("");
  await expect.poll(() => root.querySelector("[data-readout]")?.textContent).toBe("3 frames");
});
