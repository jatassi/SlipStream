import type { NodesDownloadingResource } from "./nodesDownloadingResource.ts";
import type { NodesStorageResource } from "./nodesStorageResource.ts";

export interface NodesResources {
  /** Its torrent client; absent unless the Node stands in Downloading and is in step */
  downloading?: NodesDownloadingResource;
  /** Its library roots and the volumes they sit on; absent unless the Node stands in Storage. While it is away, as it last reported them */
  storage?: NodesStorageResource;
}
