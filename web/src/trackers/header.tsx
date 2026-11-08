import type { ReactNode } from "react";

import type { TrackersStats, TrackersTrackerItem } from "../api/gen/model";
import { bytes } from "../format";
import { KeyValues } from "../routes/_app/nodes/-facts";
import { Chip } from "../ui/chip";
import { Data } from "../ui/data";
import { Details } from "../ui/details";
import { noRuleSentence, privacyLabels, ruleSentence, ruleSource, standingSentence } from "./words";

const lines = (values: readonly string[]): ReactNode =>
  values.length === 0 ? (
    <span>None</span>
  ) : (
    <span className="flex flex-col">
      {values.map((value) => (
        <Data key={value}>{value}</Data>
      ))}
    </span>
  );

function Stats({ stats }: { stats: TrackersStats }) {
  const shown: readonly (readonly [string, string])[] = [
    ["Uploaded", bytes(stats.uploaded)],
    ["Downloaded", bytes(stats.downloaded)],
    ["Ratio", stats.ratio.toFixed(2)],
    ["Torrents", String(stats.torrents)],
    ["Seeding", bytes(stats.seedingBytes)],
    ["Grabs in 30 days", String(stats.grabs30d)],
    ["Owe seeding", String(stats.owing)],
  ];
  return (
    <dl aria-label="Totals" className="flex flex-wrap gap-x-8 gap-y-3">
      {shown.map(([term, value]) => (
        <div key={term} className="flex flex-col gap-0.5">
          <dt className="text-meta text-ink-3">{term}</dt>
          <dd className="text-body text-ink-1 tabular-nums in-data-[density=touch]:text-body-touch">
            {value}
          </dd>
        </div>
      ))}
    </dl>
  );
}

export const PrivacyChip = ({ tracker }: { tracker: TrackersTrackerItem }) => (
  <Chip look="outline">{privacyLabels[tracker.privacy]}</Chip>
);

/** The rule in force and where it came from, or why there is none. */
export function RuleLine({ tracker: { name, rule, privacy } }: { tracker: TrackersTrackerItem }) {
  return (
    <p className="text-ink-1">
      {rule === null ? noRuleSentence(name, privacy) : ruleSentence(name, rule)}
      {rule !== null && <span className="text-ink-3"> {ruleSource(name, rule)}.</span>}
    </p>
  );
}

/** What a Tracker requires, where it stands with it, and its totals. */
export function TrackerHeader({
  tracker,
  stats,
}: {
  tracker: TrackersTrackerItem;
  stats: TrackersStats;
}) {
  return (
    <div className="mb-6 flex flex-col gap-4">
      <RuleLine tracker={tracker} />
      <Details summary={standingSentence(tracker.name, tracker.counts)}>
        <div className="mt-2 w-full">
          <KeyValues
            rows={[
              ["Keys", lines(tracker.keys)],
              ["Announce hosts", lines(tracker.hosts)],
            ]}
          />
        </div>
      </Details>
      <Stats stats={stats} />
    </div>
  );
}
