import { type Plane, type Projection } from "@anatomy/iso-kit";

import { CONDUIT, FEET, RISER, type Route, WORKER_SCALE } from "../layout";
import { type Drawn, type Item, block, item, mark } from "./part";
import { tube } from "./tube";

function riser(
  P: Projection,
  name: string,
  [cx, cy]: Plane,
  degrees: number,
  scale: number,
): Drawn {
  const half = (RISER * scale) / 2;
  const radians = (degrees * Math.PI) / 180;
  const along: Plane = [Math.cos(radians) * half, Math.sin(radians) * half];
  const across: Plane = [-along[1], along[0]];
  const ring: Plane[] = [
    [cx - along[0] - across[0], cy - along[1] - across[1]],
    [cx + along[0] - across[0], cy + along[1] - across[1]],
    [cx + along[0] + across[0], cy + along[1] + across[1]],
    [cx - along[0] + across[0], cy - along[1] + across[1]],
  ];
  return block(P, name, ring, 0, FEET * scale, { tone: "lo" });
}

/**
 * The Node link between the Controller and a Worker: a glass conduit on the
 * floor, standing on a riser under each tile, with the capsules that carry
 * its traffic.
 */
export function link(P: Projection, route: Route): Item {
  const name = `link.${route.worker}`;
  const conduit = tube(P, `${name}.conduit`, route.points, CONDUIT, { className: "glass conduit" });
  const [a, b] = route.points;
  return item(
    name,
    [(a[0] + b[0]) / 2, (a[1] + b[1]) / 2, 0],
    [
      riser(P, `stand.${name}.controller`, route.risers[0], route.degrees, 1),
      riser(P, `stand.${name}.worker`, route.risers[1], route.degrees, WORKER_SCALE),
      conduit,
      mark(<g data-capsules="" />),
    ],
    [conduit.pipe],
  );
}
