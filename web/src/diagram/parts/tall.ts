import type { Projection } from "@anatomy/iso-kit";

import type { NodeRow } from "../../routes/_app/nodes/-landing/view";
import { drawersOf } from "../storage-figures";
import { board } from "./board";
import { cabinet } from "./cabinet";
import type { Item } from "./part";
import { press } from "./press";
import { tower } from "./tower";

/** The tall machines, which stand along the back of the tile and to its left. */
export const TALL = {
  downloading: board,
  storage: (P: Projection, node: NodeRow): Item[] => [cabinet(P, drawersOf(node.storage))],
  converting: (P: Projection): Item[] => [press(P)],
  relaying: (P: Projection): Item[] => [tower(P)],
};
