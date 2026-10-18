import type { Projection } from "@anatomy/iso-kit";

import type { NodeRow } from "../../routes/_app/nodes/-landing/view";
import { booth } from "./booth";
import { catalog } from "./catalog";
import { dish } from "./dish";
import { mailbox } from "./mailbox";
import type { Item } from "./part";
import { television } from "./television";
import { typewriter } from "./typewriter";

/** The small machines, about a third of a piece board's floor each, in two rows in front of the tall ones. */
export const SMALL = {
  searching: (P: Projection): Item[] => [dish(P)],
  subtitles: (P: Projection): Item[] => [typewriter(P)],
  metadata: (P: Projection): Item[] => [catalog(P)],
  "media-server-link": (P: Projection): Item[] => [television(P)],
  notifying: (P: Projection): Item[] => [mailbox(P)],
  serving: (P: Projection, node: NodeRow): Item[] => [booth(P, node.listening)],
};
