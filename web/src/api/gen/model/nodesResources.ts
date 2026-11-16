import type { NodesDownloadingResource } from "./nodesDownloadingResource.ts";
import type { NodesStorageResource } from "./nodesStorageResource.ts";

export interface NodesResources {
  /** Its torrent client; absent unless the Node has the Downloading Role and is connected */
  downloading?: NodesDownloadingResource;
  /** Its library roots and the volumes they sit on; absent unless the Node has the Storage Role. While it is away, as it last reported them */
  storage?: NodesStorageResource;
}
