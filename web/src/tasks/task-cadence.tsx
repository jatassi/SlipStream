import type { ReactNode } from "react";

import { useListTasksSuspense, useUpdateTask } from "../api/gen/query/tasks/tasks";
import { SettingField } from "../ui/setting-field";

/**
 * A task's cadence, saved through its task. A task the Tasks page doesn't
 * list, such as one with no unit to run, shows fallback: its owner's own
 * field for the same stored cadence.
 */
export function TaskCadence({
  task,
  variant = "row",
  fallback = null,
}: {
  task: string;
  variant?: "row" | "full";
  fallback?: ReactNode;
}) {
  const shown = useListTasksSuspense({
    query: {
      select: ({ groups }) =>
        groups.flatMap((group) => group.tasks).find((candidate) => candidate.id === task),
    },
  }).data;
  const update = useUpdateTask();
  if (shown === undefined) {
    return fallback;
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
