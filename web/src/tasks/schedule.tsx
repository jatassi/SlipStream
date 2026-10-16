import type { TasksSchedule, TasksSchedulePatch } from "../api/gen/model";
import { useUpdateTaskSettings } from "../api/gen/query/tasks/tasks";
import { countWords, durationWords } from "../format";
import { Details } from "../ui/details";
import { Section } from "../ui/section";
import { SettingField, SettingGroup } from "../ui/setting-field";

const summary = ({ jitterPercent, jitterCap, headerDelay, partialWarnIntervals }: TasksSchedule) =>
  `Interval tasks start up to ${countWords(jitterPercent.value, jitterPercent.unit)} late, at most ${durationWords(jitterCap.value)}. A scheduled run shows in the header after ${durationWords(headerDelay.value)}. A Partial run warns once a unit has gone ${countWords(partialWarnIntervals.value, partialWarnIntervals.unit)} without success.`;

export function Schedule({ settings }: { settings: TasksSchedule }) {
  const update = useUpdateTaskSettings();
  const save = (data: TasksSchedulePatch) => update.mutateAsync({ data });
  return (
    <Section title="Schedule">
      <SettingGroup>
        <SettingField field={settings.window} onSave={(window) => save({ window })} />
        <SettingField field={settings.overdue} onSave={(overdue) => save({ overdue })} />
        <SettingField field={settings.runsKept} onSave={(runsKept) => save({ runsKept })} />
        <Details summary={summary(settings)}>
          <div className="mt-2 flex flex-col gap-5">
            <SettingField
              field={settings.jitterPercent}
              onSave={(jitterPercent) => save({ jitterPercent })}
            />
            <SettingField field={settings.jitterCap} onSave={(jitterCap) => save({ jitterCap })} />
            <SettingField
              field={settings.headerDelay}
              onSave={(headerDelay) => save({ headerDelay })}
            />
            <SettingField
              field={settings.partialWarnIntervals}
              onSave={(partialWarnIntervals) => save({ partialWarnIntervals })}
            />
          </div>
        </Details>
      </SettingGroup>
    </Section>
  );
}
