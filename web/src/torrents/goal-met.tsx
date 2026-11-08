import { Suspense } from "react";

import type {
  TrackersAct,
  TrackersKept,
  TrackersPlannedTorrent,
  TrackersWaitWhy,
} from "../api/gen/model";
import { usePreviewSeedingGoalsSuspense } from "../api/gen/query/trackers/trackers";
import { List, ListRow } from "../ui/list";
import { Section } from "../ui/section";
import { actionLabels } from "../trackers/words";
import { Skeleton } from "../ui/skeleton";
import type { Row } from "./model";
import type { Names } from "./names";

const keptWords: {
  readonly [W in TrackersKept["why"]]: (name: string, node: string) => string;
} = {
  "not-imported": (name) =>
    `SlipStream removes ${name} and keeps its download copy, because its Grab didn't import.`,
  draining: (name, node) =>
    `SlipStream removes ${name} and keeps its download copy, because ${node} is draining Downloading.`,
};

const waiting = "Goal met, waiting";
const notActing = "Goal met, not acting";
const waitGroups = [waiting, notActing] as const;

const waitWords: {
  readonly [W in TrackersWaitWhy]: {
    readonly group: (typeof waitGroups)[number];
    readonly words: string;
  };
} = {
  removing: { group: notActing, words: "It's being removed." },
  resumed: {
    group: notActing,
    words: "You resumed it after its goal was met, so it seeds with no end.",
  },
  ran: { group: notActing, words: "Its goal already acted on it." },
  incomplete: {
    group: waiting,
    words: "Its goal waits until all of its selected files are there.",
  },
  "grab-downloading": {
    group: waiting,
    words: "Its goal waits until its Grab finishes downloading.",
  },
  "grab-importing": { group: waiting, words: "Its goal waits until its Grab finishes importing." },
  reconciling: {
    group: waiting,
    words: "Its goal waits until SlipStream finishes reconciling the library after a restore.",
  },
};

type Shown<T> = T & { row: Row };
type Named = (entry: Shown<{ torrent: TrackersPlannedTorrent }>) => string;

const trackerOf = ({ torrent }: { torrent: TrackersPlannedTorrent }) =>
  torrent.tracker?.name ?? "No known Tracker";

function WouldAct({
  acts,
  kept,
  nameOf,
  names,
}: {
  acts: readonly Shown<TrackersAct>[];
  kept: readonly Shown<TrackersKept>[];
  nameOf: Named;
  names: Names;
}) {
  return (
    <Section title="Would act now" count={acts.length + kept.length}>
      {acts.length + kept.length === 0 ? (
        <p className="text-ink-3">No goal would act if Seeding goals ran now.</p>
      ) : (
        <List label="Would act now">
          {acts.map((act) => (
            <ListRow
              key={act.torrent.id}
              title={nameOf(act)}
              meta={`${trackerOf(act)} · ${actionLabels[act.action]}`}
            />
          ))}
          {kept.map((entry) => (
            <ListRow
              key={entry.torrent.id}
              title={nameOf(entry)}
              meta={`${trackerOf(entry)} · ${actionLabels.remove}`}
              detail={
                <span className="text-meta text-ink-2">
                  {keptWords[entry.why](nameOf(entry), names.node(entry.row.nodeId))}
                </span>
              }
            />
          ))}
        </List>
      )}
    </Section>
  );
}

function Preview({ rows, names }: { rows: readonly Row[]; names: Names }) {
  const preview = usePreviewSeedingGoalsSuspense().data;
  const byId = new Map(rows.map((row) => [row.id, row]));
  const shown = <T extends { torrent: TrackersPlannedTorrent }>(entries: readonly T[]) =>
    entries.flatMap((entry) => {
      const row = byId.get(entry.torrent.id);
      return row === undefined ? [] : [{ ...entry, row }];
    });
  const nameOf: Named = ({ torrent, row }) => names.work(row.workId) ?? torrent.name;
  const waits = shown(preview.waits);
  return (
    <>
      <WouldAct
        acts={shown(preview.acts)}
        kept={shown(preview.kept)}
        nameOf={nameOf}
        names={names}
      />
      {waitGroups.map((group) => {
        const held = waits.filter((wait) => waitWords[wait.why].group === group);
        return (
          held.length > 0 && (
            <Section key={group} title={group} count={held.length}>
              <List label={group}>
                {held.map((wait) => (
                  <ListRow
                    key={wait.torrent.id}
                    title={nameOf(wait)}
                    meta={waitWords[wait.why].words}
                  />
                ))}
              </List>
            </Section>
          )
        );
      })}
    </>
  );
}

/** Above the Goal met tab's table: what the next Seeding goals run would do to these rows' torrents. */
export function GoalMetPreview({ rows, names }: { rows: readonly Row[]; names: Names }) {
  return (
    <div className="mb-6">
      <Suspense fallback={<Skeleton label="Loading what the goals would do" />}>
        <Preview rows={rows} names={names} />
      </Suspense>
    </div>
  );
}
