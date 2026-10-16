import type { TasksGroup, TasksTask } from "../api/gen/model";
import { Section } from "../ui/section";
import { useShell } from "../ui/shell-context";
import { type Column, Table } from "../ui/table";
import { TaskAction } from "./task-actions";
import { TaskCadence } from "./task-cadence";
import { LastRun, type Live, TaskName, TaskState, progressOf } from "./task-cells";
import { TaskList } from "./task-list";
import { groupTitles } from "./words";

type Columns = readonly [Column<TasksTask>, ...Column<TasksTask>[]];

function columnsOf(multinode: boolean, live: Live): Columns {
  const nodes: Column<TasksTask>[] = multinode
    ? [{ id: "nodes", header: "Nodes", cell: (task) => task.nodes.join(", "), hideBelow: 980 }]
    : [];
  return [
    { id: "task", header: "Task", cell: (task) => <TaskName task={task} /> },
    { id: "cadence", header: "Cadence", cell: (task) => <TaskCadence task={task.id} /> },
    {
      id: "last",
      header: "Last run",
      cell: (task) => <LastRun run={task.lastRun} now={live.now} />,
    },
    {
      id: "next",
      header: "Next",
      cell: (task) => <TaskState task={task} progress={progressOf(task, live)} now={live.now} />,
    },
    ...nodes,
    { id: "action", header: "", align: "end", cell: (task) => <TaskAction task={task} /> },
  ];
}

export function TaskGroup({
  group: { group, tasks },
  multinode,
  live,
}: {
  group: TasksGroup;
  multinode: boolean;
  live: Live;
}) {
  const title = groupTitles[group];
  return (
    <Section title={title}>
      {useShell() === "phone" ? (
        <TaskList label={title} tasks={tasks} live={live} />
      ) : (
        <Table
          label={title}
          lines={2}
          rows={tasks}
          rowKey={(task) => task.id}
          tone={(task) => (task.state.kind === "waiting" ? "warn" : undefined)}
          columns={columnsOf(multinode, live)}
        />
      )}
    </Section>
  );
}
