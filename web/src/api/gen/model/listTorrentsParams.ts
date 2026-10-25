import type { ListTorrentsTab } from "./listTorrentsTab.ts";

export type ListTorrentsParams = {
  /**
   * Only these Torrents, such as the ones a torrents.changed named
   * @nullable
   */
  ids?: string[] | null;
  tab?: ListTorrentsTab;
  nodeId?: string;
  contentType?: string;
  /**
   * Matches the release name, the Grabs' release titles and their Works' labels, ignoring case
   * @maxLength 200
   */
  q?: string;
};
