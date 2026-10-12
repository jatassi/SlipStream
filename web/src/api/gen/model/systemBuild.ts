export interface SystemBuild {
  commit?: string;
  fingerprint?: string;
  libtorrent?: string;
  nativeBundle?: string;
  stamped: boolean;
  target?: string;
  version?: string;
}
