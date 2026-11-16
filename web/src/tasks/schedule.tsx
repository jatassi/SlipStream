import { useQueryClient } from "@tanstack/react-query";

import type { TasksSchedule, TasksSchedulePatch } from "../api/gen/model";
import { useUpdateTaskSettings } from "../api/gen/query/tasks/tasks";
import { countWords, durationWords } from "../format";
import { adoptTaskSettings } from "../live/adopt";
import { Details } from "../ui/details";
import { Section } from "../ui/section";
import { SettingField, SettingGroup } from "../ui/setting-field";

const summary = ({ jitterPercent, jitterCap, headerDelay, partialWarnIntervals }: TasksSchedule) =>
  `Tasks that repeat on an interval start up to ${countWords(jitterPercent.value, jitterPercent.unit)} late, at most ${durationWords(jitterCap.value)}. The top bar shows a scheduled run after ${durationWords(headerDelay.value)}. Health warns when one of a task's units hasn't succeeded for ${countWords(partialWarnIntervals.value, partialWarnIntervals.unit)}.`;

export function Schedule({ settings }: { settings: TasksSchedule }) {
  const queryClient = useQueryClient();
  const update = useUpdateTaskSettings();
  const save = async (data: TasksSchedulePatch) => {
    adoptTaskSettings(queryClient, await update.mutateAsync({ data }));
  };
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
