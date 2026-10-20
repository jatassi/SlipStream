export interface DownloadclientsLiveRow {
  /** Bytes downloaded */
  done: number;
  /** Payload bytes a second it downloads */
  downBps: number;
  id: string;
  peers: number;
  seeds: number;
  /** Payload bytes a second it uploads */
  upBps: number;
  /** Bytes its selected files hold */
  wanted: number;
}
