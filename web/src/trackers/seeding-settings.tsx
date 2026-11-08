import { useQueryClient } from "@tanstack/react-query";

import {
  setDefaultGoal,
  updateSeedingSettings,
  useGetDefaultGoalSuspense,
  useGetSeedingSettingsSuspense,
} from "../api/gen/query/trackers/trackers";
import { adoptDefaultGoal, adoptSeedingSettings } from "../live/adopt-trackers";
import { KeyValues } from "../routes/_app/nodes/-facts";
import { TaskCadence } from "../tasks/task-cadence";
import { Section } from "../ui/section";
import { SettingField, SettingGroup } from "../ui/setting-field";
import { toast } from "../ui/toast";
import { GoalEditor } from "./goal-editor";

export function DefaultGoal() {
  const queryClient = useQueryClient();
  const goal = useGetDefaultGoalSuspense().data;
  return (
    <Section title="Default goal">
      <div className="flex flex-col gap-4">
        <p className="text-ink-2">
          Every Tracker follows this goal until you give it its own on its Seeding tab. A goal never
          ends a torrent&apos;s seeding before its Tracker&apos;s rule is met.
        </p>
        <GoalEditor
          key={JSON.stringify(goal)}
          goal={goal}
          onSave={async (body) => {
            adoptDefaultGoal(queryClient, await setDefaultGoal(body));
            toast("Saved the default goal");
          }}
        />
      </div>
    </Section>
  );
}

export function RuleSettings() {
  const queryClient = useQueryClient();
  const { settings } = useGetSeedingSettingsSuspense().data;
  return (
    <Section title="Rules and warnings">
      <div className="flex flex-col gap-3">
        <p className="text-ink-2">
          A rule&apos;s seeding time is met only once the margin past it has passed too, in case a
          Tracker counts its time a little differently. Health warns when a torrent that still owes
          seeding hasn&apos;t announced for the warning time.
        </p>
        <SettingGroup>
          {settings.map((field) => (
            <SettingField
              key={field.key}
              field={field}
              onSave={async (value) => {
                adoptSeedingSettings(
                  queryClient,
                  await updateSeedingSettings({ values: { [field.key]: value } }),
                );
              }}
            />
          ))}
        </SettingGroup>
        <p className="text-ink-2">
          The Seeding goals task reads each Tracker&apos;s rule and records what each of its
          torrents still owes.
        </p>
        <KeyValues rows={[["Seeding goals", <TaskCadence key="goals" task="seeding-goals" />]]} />
      </div>
    </Section>
  );
}
