export interface RootsRoot {
  /** @pattern ^[a-z][a-z0-9]{1,15}$ */
  contentType: string;
  id: string;
  /** The Node whose disk holds it */
  nodeId: string;
  path: string;
}
