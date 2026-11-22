import type { GrabsGrab } from "../api/gen/model";
import type { WorkNames } from "../torrents/acting";
import type { Row } from "../torrents/model";

/**
 * A Grab as its parts draw it: its Work's name, its Torrent while the torrent
 * list holds it, and the names of the Works a torrent's files belong to.
 */
export type Job = {
  readonly grab: GrabsGrab;
  readonly work: string;
  readonly torrent: Row | undefined;
  readonly names: WorkNames;
};
