import * as A from "../vendor/anatomy/kit/audit.mjs";
import { fixtureClock, fixtureOf, FIXTURE_SIZES } from "../src/diagram/fixtures";
import { drawingOf } from "../src/diagram/geometry";
import { landingOf } from "../src/routes/_app/nodes/-landing/view";
import { recorderFor, record } from "./diagram-record";

// Runs the anatomy audit on each fixture's drawing with exactly the order the
// page paints it in: the links, then the tiles back to front, each tile's
// parts in depth-key order. Exits non-zero unless every fixture passes.

const LINKS = -1e6;
const TILE_STEP = 1e5;

const sizes = process.argv.slice(2).map(Number).filter((size) => size > 0);
let failed = 0;
for (const size of sizes.length > 0 ? sizes : FIXTURE_SIZES) {
  const fixture = fixtureOf(size);
  const drawing = drawingOf(landingOf(fixture.topology, fixture.board, new Set(), fixtureClock));
  const { P } = drawing.stage;
  const R = recorderFor(P);
  drawing.links.forEach(({ item }, index) => record(R, item, { key: LINKS + index }));
  drawing.tiles.forEach(({ placement, items }, rank) => {
    const [x, y] = placement.at;
    items.forEach((item, index) =>
      record(R, item, { by: [x, y, 0], scale: placement.scale, key: rank * TILE_STEP + index }),
    );
  });
  R.settled = {
    items: R.items,
    overlaps: A.overlapsOf(R.items, P, { states: [0], skip: A.sameRun }),
    solved: { keys: R.items.map(({ key }) => key), rules: 0, cycles: [], dropped: [] },
    states: [0],
  };
  console.log(`fixture ${String(size)}:`);
  A.auditOrExit(R, P, { argv: ["--audit"] });
  failed += process.exitCode === 1 ? 1 : 0;
  process.exitCode = 0;
}
if (failed > 0) {
  console.error(`audit failed on ${String(failed)} fixture${failed === 1 ? "" : "s"}`);
  process.exit(1);
}
