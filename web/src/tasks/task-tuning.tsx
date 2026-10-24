import type { SettingField as Field, TasksTask } from "../api/gen/model";
import { useUpdateTask } from "../api/gen/query/tasks/tasks";
import { Section } from "../ui/section";
import { SettingField, SettingGroup } from "../ui/setting-field";
import { TaskCadence } from "./task-cadence";

export function TaskTuning({ task, tuning }: { task: TasksTask; tuning: readonly Field[] }) {
  const update = useUpdateTask();
  return (
    <Section title="When it runs">
      <SettingGroup>
        <div className="flex flex-col gap-3">
          <p className="text-(length:--body-size) text-ink-1">{task.cadence.words}</p>
          <TaskCadence task={task.id} variant="full" />
        </div>
        <SettingField
          field={task.overdue}
          onSave={(overdue) => update.mutateAsync({ taskId: task.id, data: { overdue } })}
        />
        {tuning.map((field) => (
          <SettingField
            key={field.key}
            field={field}
            onSave={(value) =>
              update.mutateAsync({ taskId: task.id, data: { tuning: { [field.key]: value } } })
            }
          />
        ))}
      </SettingGroup>
    </Section>
  );
}
