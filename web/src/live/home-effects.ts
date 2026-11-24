import type { QueryClient } from "@tanstack/react-query";

import type { HomeSummary } from "../api/gen/model";
import { getGetTorrentCountsQueryKey } from "../api/gen/query/downloadclients/downloadclients";
import { getGetHomeQueryKey, getGetHomeSettingsQueryKey } from "../api/gen/query/home/home";
import { type LiveEffect, type LiveEffects, type ProductEventName, both } from "./types";

const home = { queryKey: getGetHomeQueryKey() };

const readHome =
  <N extends ProductEventName>(): LiveEffect<N> =>
  (_data, { refresh }) => {
    refresh(home);
  };

/** The Torrents of the Jobs a cached Home shows. */
function shownTorrents(queryClient: QueryClient): Set<string> {
  const ids = new Set<string>();
  for (const [, summary] of queryClient.getQueriesData<HomeSummary>(home)) {
    for (const { torrent } of summary?.jobs.rows ?? []) {
      if (torrent !== null) {
        ids.add(torrent.id);
      }
    }
  }
  return ids;
}

/**
 * Home reads its summary again for a change to what it shows: a Grab's
 * stage, an Import, a Work, a client's or Node's reach, Reconciling, a prune
 * of History, a Move, or Rows on Home. A progress tick changes no row's
 * place, and a Torrent's change reads the summary only when a shown Job
 * holds it, so its menu offers what the Torrent can do now.
 */
export const homeEffects = {
  "home.settings-changed": (_data, { refresh }) => {
    refresh({ queryKey: getGetHomeSettingsQueryKey() });
    refresh(home);
  },
  "grabs.changed": ({ progress }, { refresh }) => {
    if (!progress) {
      refresh(home);
    }
  },
  "imports.changed": readHome(),
  "files.changed": readHome(),
  "works.changed": readHome(),
  "works.removed": readHome(),
  "nodes.changed": readHome(),
  "reconciling.changed": readHome(),
  "history.changed": readHome(),
  "moves.changed": readHome(),
  "download-clients.changed": ({ clientIds }, { refresh }) => {
    if (clientIds.length > 0) {
      refresh(home);
    }
  },
  "torrents.changed": ({ ids }, { queryClient, refresh }) => {
    refresh({ queryKey: getGetTorrentCountsQueryKey() });
    const shown = shownTorrents(queryClient);
    if (ids.some((id) => shown.has(id))) {
      refresh(home);
    }
  },
} satisfies Partial<LiveEffects>;

/** The table with Home's effects run after its own. */
export const withHome = (table: Omit<LiveEffects, "home.settings-changed">): LiveEffects => ({
  ...table,
  "home.settings-changed": homeEffects["home.settings-changed"],
  "grabs.changed": both(table["grabs.changed"], homeEffects["grabs.changed"]),
  "imports.changed": both(table["imports.changed"], homeEffects["imports.changed"]),
  "files.changed": both(table["files.changed"], homeEffects["files.changed"]),
  "works.changed": both(table["works.changed"], homeEffects["works.changed"]),
  "works.removed": both(table["works.removed"], homeEffects["works.removed"]),
  "nodes.changed": both(table["nodes.changed"], homeEffects["nodes.changed"]),
  "reconciling.changed": both(table["reconciling.changed"], homeEffects["reconciling.changed"]),
  "history.changed": both(table["history.changed"], homeEffects["history.changed"]),
  "moves.changed": both(table["moves.changed"], homeEffects["moves.changed"]),
  "download-clients.changed": both(
    table["download-clients.changed"],
    homeEffects["download-clients.changed"],
  ),
  "torrents.changed": both(table["torrents.changed"], homeEffects["torrents.changed"]),
});
