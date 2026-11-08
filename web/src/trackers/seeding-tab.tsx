import { useQueryClient } from "@tanstack/react-query";

import type { TrackersTracker, TrackersTrackerItem } from "../api/gen/model";
import { clearTrackerRule, setTrackerRule } from "../api/gen/query/trackers/trackers";
import { adoptTracker } from "../live/adopt-trackers";
import { Section } from "../ui/section";
import { toast } from "../ui/toast";
import { GoalSection } from "./goal-section";
import { RuleLine } from "./header";
import { RuleEditor } from "./rule-editor";
import { UseInstead } from "./use-instead";

function RuleSection({
  tracker,
  adopt,
}: {
  tracker: TrackersTrackerItem;
  adopt: (saved: TrackersTracker) => void;
}) {
  const { id, name, rule } = tracker;
  return (
    <Section title="Rule">
      <div className="flex flex-col gap-4">
        <RuleLine tracker={tracker} />
        {tracker.privacy !== "public" && (
          <RuleEditor
            key={JSON.stringify(rule)}
            rule={rule}
            onSave={async (body) => {
              adopt(await setTrackerRule(id, body));
              toast(`Saved ${name}'s rule`);
            }}
            aside={
              rule?.source === "admin" && (
                <UseInstead
                  label={`Use what ${name} states`}
                  drop={async () => {
                    adopt(await clearTrackerRule(id));
                  }}
                />
              )
            }
          />
        )}
      </div>
    </Section>
  );
}

/** What a Tracker requires, which the Admin may correct, and the goal its torrents seed to. */
export function SeedingTab({ tracker }: { tracker: TrackersTrackerItem }) {
  const queryClient = useQueryClient();
  const adopt = (saved: TrackersTracker) => {
    adoptTracker(queryClient, saved);
  };
  return (
    <div className="max-w-[640px]">
      <RuleSection tracker={tracker} adopt={adopt} />
      <GoalSection tracker={tracker} adopt={adopt} />
    </div>
  );
}
