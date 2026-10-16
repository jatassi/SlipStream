import type { TasksUnit } from "../api/gen/model";
import { sinceWords } from "../format";
import { Data } from "../ui/data";
import { Section } from "../ui/section";
import { type Column, Table } from "../ui/table";
import { OutcomeChip } from "./outcome-chip";
import { unitsAnchor } from "./task-head";

const causeOf = (unit: TasksUnit) =>
  unit.dueOnReturn ? `${unit.cause ?? "Skipped"}. Runs when ${unit.node} returns.` : unit.cause;

function columnsOf(now: Date): readonly [Column<TasksUnit>, ...Column<TasksUnit>[]] {
  return [
    { id: "unit", header: "Unit", cell: (unit) => unit.label },
    { id: "node", header: "Node", cell: (unit) => unit.node },
    {
      id: "outcome",
      header: "Last outcome",
      cell: (unit) =>
        unit.outcome === null ? (
          <span className="text-ink-3">Not run yet</span>
        ) : (
          <OutcomeChip outcome={unit.outcome} />
        ),
    },
    {
      id: "success",
      header: "Last success",
      cell: (unit) =>
        unit.lastSuccessAt === null ? (
          <span className="text-ink-3">Never</span>
        ) : (
          <Data>{sinceWords(unit.lastSuccessAt, now)}</Data>
        ),
    },
    {
      id: "cause",
      header: "Cause",
      hideBelow: 760,
      cell: (unit) => <span className="text-ink-2">{causeOf(unit)}</span>,
    },
  ];
}

export function TaskUnits({ units, now }: { units: readonly TasksUnit[]; now: Date }) {
  return (
    <Section id={unitsAnchor} title="Units" count={units.length}>
      <Table
        label="Units"
        rows={units}
        rowKey={(unit) => unit.key}
        stackBelow={560}
        tone={(unit) => (unit.dueOnReturn ? "warn" : undefined)}
        columns={columnsOf(now)}
      />
    </Section>
  );
}
