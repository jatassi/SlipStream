export interface WorksWorkRoot {
  id: string;
  /** The Storage Node whose disk holds it */
  nodeId: string;
  /** The folder's path on that Node, in that Node's format */
  path: string;
}
