import type { NodesDownloadingResource } from "./nodesDownloadingResource.ts";

export interface NodesResources {
  /** Its torrent client; absent unless the Node stands in Downloading and is in step */
  downloading?: NodesDownloadingResource;
}
