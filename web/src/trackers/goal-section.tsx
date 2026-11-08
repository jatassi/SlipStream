import type { TrackersTracker, TrackersTrackerItem } from "../api/gen/model";
import {
  clearTrackerGoal,
  setTrackerGoal,
  useGetDefaultGoalSuspense,
} from "../api/gen/query/trackers/trackers";
import { Section } from "../ui/section";
import { toast } from "../ui/toast";
import { GoalEditor } from "./goal-editor";
import { UseInstead } from "./use-instead";

/** A Tracker's own goal, or the default goal it follows until it has one. */
export function GoalSection({
  tracker,
  adopt,
}: {
  tracker: TrackersTrackerItem;
  adopt: (saved: TrackersTracker) => void;
}) {
  const defaultGoal = useGetDefaultGoalSuspense().data;
  const { id, name, goal } = tracker;
  return (
    <Section title="Goal">
      <div className="flex flex-col gap-4">
        <p className="text-ink-2">
          {goal === null
            ? `${name} follows the default goal from Settings > Trackers. Saving a different goal here gives ${name} its own.`
            : `${name} has its own goal. A goal never ends a torrent's seeding before ${name}'s rule is met.`}
        </p>
        <GoalEditor
          key={JSON.stringify([goal, defaultGoal])}
          goal={goal ?? defaultGoal}
          onSave={async (body) => {
            adopt(await setTrackerGoal(id, body));
            toast(`Saved ${name}'s goal`);
          }}
          aside={
            goal !== null && (
              <UseInstead
                label="Use the default goal"
                drop={async () => {
                  adopt(await clearTrackerGoal(id));
                }}
              />
            )
          }
        />
      </div>
    </Section>
  );
}
