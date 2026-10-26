import type { DownloadclientsClientCard, RootsVolumeList } from "../../api/gen/model";

/**
 * Whether a Node's downloads folder sits on another drive than every library
 * root it holds, so each import copies the whole file. Both mounts are as the
 * Node names its volumes, so the compare needs no path rules.
 */
export function downloadsApart(card: DownloadclientsClientCard, volumes: RootsVolumeList): boolean {
  const roots = volumes.nodes.find((node) => node.nodeId === card.nodeId)?.volumes ?? [];
  const { downloads } = card;
  return (
    downloads !== undefined &&
    roots.length > 0 &&
    !roots.some((volume) => volume.mount === downloads.mount)
  );
}
