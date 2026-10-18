export interface RootsRoot {
  /** @pattern ^[a-z][a-z0-9]{1,15}$ */
  contentType: string;
  /** Where its Content Type's new Works go */
  default: boolean;
  /** The bytes of its files */
  fileBytes: number;
  id: string;
  /** The Node whose disk holds it */
  nodeId: string;
  path: string;
  /** The Works that live in it */
  works: number;
}
