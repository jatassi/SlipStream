export interface NodesDialing {
  address: string;
  since: string;
  /** The Node hasn't paired within timings.nodes.dial_report of the paste */
  unreachable: boolean;
}
