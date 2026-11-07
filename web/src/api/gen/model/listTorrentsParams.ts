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
   * Only one client's Torrents: an external client's id, or built-in for every Node's built-in client
   * @pattern ^(built-in|[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12})$
   */
  client?: string;
  /**
   * Matches the release name, the Grabs' release titles and their Works' labels, ignoring case
   * @maxLength 200
   */
  q?: string;
};
