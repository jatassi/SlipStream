import type { DownloadclientsRefusalCode } from "./downloadclientsRefusalCode.ts";

export interface DownloadclientsRefusal {
  /** window: when every announce URL accepts an announce */
  acceptsAt?: string;
  code: DownloadclientsRefusalCode;
  /** importing: the Work whose import holds it */
  workId?: string;
}
