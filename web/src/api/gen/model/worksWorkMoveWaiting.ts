/**
 * What it waits for; absent while it waits for nothing
 */
export type WorksWorkMoveWaiting =
  | "pair_busy"
  | "low_priority"
  | "node_offline"
  | "no_route"
  | "space"
  | "renames"
  | "downloading"
  | "importing"
  | "open_files"
  | "playing"
  | "media_servers"
  | "guard"
  | "linked_torrent"
  | "reconciling";
