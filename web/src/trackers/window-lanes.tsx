import type { TrackersWindow } from "../api/gen/model";
import { useListTorrentsSuspense } from "../api/gen/query/downloadclients/downloadclients";
import { lanesOf } from "../torrents/seeding";
import { LaneMeter } from "../ui/lane-meter";

/** A window's lanes, drawn from the row of its first live Torrent the list holds. */
export function useLanes() {
  const rows = new Map(useListTorrentsSuspense().data.rows.map((row) => [row.id, row]));
  return (window: TrackersWindow) => {
    const row = window.torrents.map((id) => rows.get(id)).find((found) => found !== undefined);
    return row === undefined ? undefined : (
      <LaneMeter lanes={lanesOf(row.seeding)} risk={row.seeding.atRisk === "yes"} />
    );
  };
}
