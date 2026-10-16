import { Link } from "@tanstack/react-router";

import type { TasksTask } from "../api/gen/model";
import { Icon } from "../ui/icon";
import { List, ListRow } from "../ui/list";
import { signals } from "../ui/signal-chip";
import { TaskAction } from "./task-actions";
import { type Live, TaskState, progressOf } from "./task-cells";
import { outcomes } from "./words";

function LastWord({ task }: { task: TasksTask }) {
  const result = task.lastRun?.result;
  if (result === undefined || result === null) {
    return null;
  }
  const { word, signal } = outcomes[result];
  return (
    <>
      {signal !== undefined && <Icon icon={signals[signal].icon} className="size-3.5 shrink-0" />}
      <span className="shrink-0">{word}</span>
      <span aria-hidden="true" className="text-ink-4">
        ·
      </span>
    </>
  );
}

export function TaskList({
  label,
  tasks,
  live,
}: {
  label: string;
  tasks: readonly TasksTask[];
  live: Live;
}) {
  return (
    <List label={label}>
      {tasks.map((task) => (
        <ListRow
          key={task.id}
          title={
            <Link to="/tasks/$taskId" params={{ taskId: task.id }}>
              {task.name}
            </Link>
          }
          meta={
            <span className="flex min-w-0 items-center gap-1.5">
              {task.state.kind === "idle" && <LastWord task={task} />}
              <span className="min-w-0 truncate">
                <TaskState task={task} progress={progressOf(task, live)} now={live.now} />
              </span>
            </span>
          }
          trailing={<TaskAction task={task} />}
        />
      ))}
    </List>
  );
}
