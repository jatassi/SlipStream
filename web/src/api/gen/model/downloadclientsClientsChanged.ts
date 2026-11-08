export interface DownloadclientsClientsChanged {
  /** The external clients whose settings or state changed */
  clientIds: string[];
  /** The Nodes whose card or settings changed */
  nodeIds: string[];
}
