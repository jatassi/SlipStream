import type { TrackersRoute } from "./trackersRoute.ts";

export interface TrackersRouteList {
  /** One for each home Node of the Tracker's Indexers, by Node name; empty unless the Tracker is private */
  routes: TrackersRoute[];
}
