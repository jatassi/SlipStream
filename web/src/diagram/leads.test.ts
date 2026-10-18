import { expect, test } from "vitest";

import { landingOf } from "../routes/_app/nodes/-landing/view";
import { drawingOf } from "./drawing";
import { fixtureClock, fixtureOf } from "./fixtures";
import { TOP } from "./layout";

const landingFrom = (count: number) => {
  const { topology, board } = fixtureOf(count);
  return landingOf(topology, board, new Set(), fixtureClock);
};

test("every machine is cabled into its tile: one cable per Role, dropping onto a grommet set on the tile's top", () => {
  for (const { node, items } of drawingOf(landingFrom(12)).tiles) {
    for (const { role } of node.roles) {
      const own = items.filter((item) => item.role === role);
      const cables = own
        .flatMap(({ pipes }) => pipes)
        .filter(({ name }) => name === `${role}.lead.cable`);
      expect(cables, role).toHaveLength(1);
      const grommet = own
        .flatMap(({ shapes }) => shapes)
        .find(({ name }) => name === `${role}.lead.grommet`);
      expect(grommet?.kind, role).toBe("disc");
      if (grommet?.kind !== "disc") {
        continue;
      }
      const end = cables[0]?.points.at(-1);
      expect(grommet.frame.o[2] + grommet.s0, role).toBe(TOP);
      expect(end?.[2], role).toBeCloseTo(grommet.frame.o[2] + grommet.s1);
      expect(
        Math.hypot((end?.[0] ?? 0) - grommet.frame.o[0], (end?.[1] ?? 0) - grommet.frame.o[1]),
        role,
      ).toBeCloseTo(0);
    }
  }
});
