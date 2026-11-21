import type { TasksUnit } from "../api/gen/model";
import type { Clock } from "../ui/clock";
import { PlainTime } from "../ui/relative-time";
import { Section } from "../ui/section";
import { type Column, Table } from "../ui/table";
import { NodeName } from "./node-name";
import { OutcomeChip } from "./outcome-chip";
import { unitsAnchor } from "./task-head";

const causeOf = (unit: TasksUnit) =>
  unit.dueOnReturn
    ? `${unit.cause ?? "Skipped"}. This unit runs when ${unit.node} reconnects.`
    : unit.cause;

function columnsOf(clock: Clock): readonly [Column<TasksUnit>, ...Column<TasksUnit>[]] {
  return [
    { id: "unit", header: "Unit", cell: (unit) => unit.label },
    {
      id: "node",
      header: "Node",
      cell: (unit) => <NodeName name={unit.node} nodeId={unit.nodeId} removed={unit.nodeRemoved} />,
    },
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
          <PlainTime at={unit.lastSuccessAt} form="clock" zone={clock.zone} data />
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

export function TaskUnits({ units, clock }: { units: readonly TasksUnit[]; clock: Clock }) {
  return (
    <Section id={unitsAnchor} title="Units" count={units.length}>
      <Table
        label="Units"
        rows={units}
        rowKey={(unit) => unit.key}
        stackBelow={560}
        tone={(unit) => (unit.dueOnReturn ? "warn" : undefined)}
        columns={columnsOf(clock)}
      />
    </Section>
  );
}
