import type { DownloadclientsRefusalCode } from "./downloadclientsRefusalCode.ts";

export interface DownloadclientsRefusal {
  /** window: when every tracker allows another announce */
  acceptsAt?: string;
  /** client_away and client_off: the external client's name; moving: the external client it moves to, absent for the built-in client */
  client?: string;
  code: DownloadclientsRefusalCode;
  /** client_away and client_off: the name of the external client's Node; moving: the Node it moves to, absent for its own Node */
  node?: string;
  /** importing: the Work whose import holds it; use_not_this_release: the Work whose Pending import waits on it */
  workId?: string;
}
