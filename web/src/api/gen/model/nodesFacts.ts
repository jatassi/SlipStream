export interface NodesFacts {
  appliedVersion: string;
  assigned: string[];
  /** The address the Node's web server bound; empty until it binds */
  httpAddress: string;
  restarting: string[];
  running: string[];
}
