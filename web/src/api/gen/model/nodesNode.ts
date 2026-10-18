import type { NodesDialView } from "./nodesDialView.ts";
import type { NodesFacts } from "./nodesFacts.ts";
import type { NodesLink } from "./nodesLink.ts";
import type { NodesNodeStanding } from "./nodesNodeStanding.ts";
import type { NodesRoleView } from "./nodesRoleView.ts";

export interface NodesNode {
  addresses: string[];
  dial: NodesDialView;
  /** Absent while the Node has no session */
  facts?: NodesFacts;
  id: string;
  /** @pattern ^[0-9a-f]{32}$ */
  intentVersion: string;
  /** The key's fingerprint */
  key: string;
  link: NodesLink;
  name: string;
  pairedAt: string;
  /** Every Role the Node stands in, in table order */
  roles: NodesRoleView[];
  self: boolean;
  /** How the Node stands with the Controller since link.since: in step, reconnecting within the blip, offline, lapsed past the offline grace, or on another version */
  standing: NodesNodeStanding;
}
