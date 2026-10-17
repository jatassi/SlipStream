import type { NodesFacts } from "./nodesFacts.ts";
import type { NodesLink } from "./nodesLink.ts";
import type { NodesRoleView } from "./nodesRoleView.ts";

export interface NodesNode {
  addresses: string[];
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
}
