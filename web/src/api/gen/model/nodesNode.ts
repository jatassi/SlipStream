import type { NodesDialView } from "./nodesDialView.ts";
import type { NodesFacts } from "./nodesFacts.ts";
import type { NodesLink } from "./nodesLink.ts";
import type { NodesNodeStanding } from "./nodesNodeStanding.ts";
import type { NodesProbe } from "./nodesProbe.ts";
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
  /** The last verdict of each probe the Node ran, kept while it is away */
  probes: NodesProbe[];
  /** Every Role the Node has, in the order of the Roles table */
  roles: NodesRoleView[];
  self: boolean;
  /** The Node's connection to the Controller since link.since: in-step (connected), reconnecting (a short drop), offline, lapsed (offline longer than the offline grace) or mismatched (another version) */
  standing: NodesNodeStanding;
}
