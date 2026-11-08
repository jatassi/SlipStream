import { Link } from "@tanstack/react-router";

import type { TrackersTrackerItem } from "../api/gen/model";
import { durationWords } from "../format";
import { AlertDot } from "../ui/count";
import { List, ListRow } from "../ui/list";
import { wantsRule } from "./sections";
import { privacyLabels, ratioWords } from "./words";

/** "Rule 10 days", "Rule ratio 1.0 or 10 days", or why there is none. */
function ruleWords({ rule, privacy }: TrackersTrackerItem): string {
  if (rule === null) {
    return privacy === "public" ? "No rule" : "No rule known";
  }
  const ratio = rule.ratio === null ? "" : `ratio ${ratioWords(rule.ratio)}`;
  const time = rule.seedSeconds === null ? "" : durationWords(rule.seedSeconds);
  const joined =
    ratio !== "" && time !== ""
      ? `${ratio} ${rule.clears === "both" ? "and" : "or"} ${time}`
      : ratio || time;
  return `Rule ${joined}`;
}

const plural = (count: number, one: string, many: string) => `${count} ${count === 1 ? one : many}`;

function Standing({ tracker }: { tracker: TrackersTrackerItem }) {
  const { torrents, owing, atRisk } = tracker.counts;
  const warn = atRisk > 0 || wantsRule(tracker);
  return (
    <span className="flex items-center gap-2 text-meta tabular-nums">
      {plural(torrents, "torrent", "torrents")}
      {owing > 0 && <span>· {owing} owe seeding</span>}
      {warn && (
        <AlertDot
          tone="warn"
          words={
            atRisk > 0
              ? `${plural(atRisk, "torrent owes", "torrents owe")} seeding and ${atRisk === 1 ? "isn't" : "aren't"} seeding`
              : "No rule known"
          }
        />
      )}
    </span>
  );
}

/** Every Tracker, each a link to its page. */
export function TrackerList({ trackers }: { trackers: readonly TrackersTrackerItem[] }) {
  return (
    <List label="Trackers">
      {trackers.map((tracker) => (
        <ListRow
          key={tracker.id}
          title={tracker.name}
          meta={[
            privacyLabels[tracker.privacy],
            ruleWords(tracker),
            tracker.goal === null ? "Default goal" : "Its own goal",
          ].join(" · ")}
          trailing={<Standing tracker={tracker} />}
          render={<Link to="/settings/trackers/$trackerId" params={{ trackerId: tracker.id }} />}
        />
      ))}
    </List>
  );
}
