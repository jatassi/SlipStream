import { useListTasksSuspense, useUpdateTask } from "../api/gen/query/tasks/tasks";
import { SettingField } from "../ui/setting-field";

/**
 * A task's cadence, saved to the task's own row. Every surface that edits a
 * cadence renders this, so tasks.changed brings each the same value.
 */
export function TaskCadence({ task, variant = "row" }: { task: string; variant?: "row" | "full" }) {
  const shown = useListTasksSuspense({
    query: {
      select: ({ groups }) =>
        groups.flatMap((group) => group.tasks).find((candidate) => candidate.id === task),
    },
  }).data;
  const update = useUpdateTask();
  if (shown === undefined) {
    return null;
  }
  const save = (cadence: number | null) => update.mutateAsync({ taskId: task, data: { cadence } });
  return variant === "row" ? (
    <SettingField
      field={shown.cadence.field}
      onSave={save}
      variant="row"
      words={shown.cadence.words}
    />
  ) : (
    <SettingField field={shown.cadence.field} onSave={save} />
  );
}
