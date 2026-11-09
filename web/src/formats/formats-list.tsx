import { Link } from "@tanstack/react-router";

import type { FormatsCustomFormat, FormatsSchema } from "../api/gen/model";
import { Chip } from "../ui/chip";
import { SignalChip } from "../ui/signal-chip";
import { type Column, Table } from "../ui/table";
import { matchSentence } from "./condition-words";
import { conditionsOf, kindsIn } from "./draft";
import { FormatMenu } from "./format-actions";
import { appWords } from "./paste-words";

function NameCell({ format }: { format: FormatsCustomFormat }) {
  return (
    <span className="flex flex-wrap items-center gap-2">
      <Link
        to="/settings/custom-formats/$formatId"
        params={{ formatId: format.id }}
        className="font-medium hover:underline"
      >
        {format.name}
      </Link>
      {format.dialect !== "slipstream" && <Chip>{appWords[format.dialect]}</Chip>}
      {format.slowReleases > 0 && <SignalChip kind="warn">Slow pattern</SignalChip>}
    </span>
  );
}

const scoredByOf = (format: FormatsCustomFormat): string =>
  format.scoredBy.length === 0 ? "None" : format.scoredBy.map((target) => target.name).join(", ");

function columnsOf(
  schema: FormatsSchema,
): readonly [Column<FormatsCustomFormat>, ...Column<FormatsCustomFormat>[]] {
  return [
    { id: "name", header: "Name", cell: (format) => <NameCell format={format} /> },
    {
      id: "matches",
      header: "Matches",
      cell: (format) => (
        <span className="line-clamp-2 text-ink-2">
          {matchSentence(conditionsOf(format, kindsIn(schema, format.dialect)))}
        </span>
      ),
    },
    { id: "scored", header: "Scored by", cell: scoredByOf },
    {
      id: "actions",
      header: "Actions",
      align: "end",
      cell: (format) => <FormatMenu format={format} />,
    },
  ];
}

export function FormatsList({
  formats,
  schema,
}: {
  formats: readonly FormatsCustomFormat[];
  schema: FormatsSchema;
}) {
  return (
    <Table
      label="Custom Formats"
      columns={columnsOf(schema)}
      rows={formats}
      rowKey={(format) => format.id}
      lines={2}
      stackBelow={560}
      tone={(format) => (format.slowReleases > 0 ? "warn" : undefined)}
    />
  );
}
