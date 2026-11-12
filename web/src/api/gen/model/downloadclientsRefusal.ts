import type { DownloadclientsRefusalCode } from "./downloadclientsRefusalCode.ts";

export interface DownloadclientsRefusal {
  /** window: when every announce URL accepts an announce */
  acceptsAt?: string;
  /** client_away and client_off: the external client's name */
  client?: string;
  code: DownloadclientsRefusalCode;
  /** client_away and client_off: the name of the external client's Node */
  node?: string;
  /** importing: the Work whose import holds it; use_not_this_release: the Work whose Pending import waits on it */
  workId?: string;
}
