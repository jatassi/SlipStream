export interface DownloadclientsAggregate {
  /** There are downloads and every one is paused */
  allPaused: boolean;
  /** Bytes the downloads hold, live while sampled */
  done: number;
  /** Bytes the downloads want */
  wanted: number;
}
