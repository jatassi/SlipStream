/**
 * Why it is weak or fails; absent when it fits
 */
export type NodesProbeReason =
  | "engine_not_listening"
  | "downloads_not_writable"
  | "downloads_on_network_share"
  | "data_not_writable"
  | "http_not_listening";
