import type { ReactElement } from "react";

import { Button } from "../../../../ui/button";
import { Chip } from "../../../../ui/chip";
import { dataFont } from "../../../../ui/data";
import { Fact, Facts } from "../../../../ui/facts";
import { PlainTime } from "../../../../ui/relative-time";
import { SignalChip } from "../../../../ui/signal-chip";
import { toast } from "../../../../ui/toast";
import type { Level, LogRow } from "./buffer";

type Line = Extract<LogRow, { kind: "line" }>;
type Divider = Extract<LogRow, { kind: "divider" }>;

const levels: { readonly [L in Level]: ReactElement } = {
  error: <SignalChip kind="fail">Error</SignalChip>,
  warn: <SignalChip kind="warn">Warn</SignalChip>,
  info: <Chip>Info</Chip>,
  debug: <Chip>Debug</Chip>,
};

const valueText = (value: unknown): string =>
  typeof value === "object" && value !== null ? JSON.stringify(value) : String(value);

async function copyLine(line: Line) {
  const { time, level, subsystem, msg, attrs } = line;
  // Spread twice, the record's fields lead and win over attributes of the same name.
  const record = { time, level, subsystem, msg };
  try {
    await navigator.clipboard.writeText(
      JSON.stringify({ ...record, ...attrs, ...record }, null, 2),
    );
  } catch {
    toast("Couldn't copy the line", { kind: "fail" });
    return;
  }
  toast("Line copied as JSON");
}

function Attributes({ line }: { line: Line }) {
  return (
    <div className="flex flex-col items-start gap-2 px-3 pt-1 pb-3">
      <Facts look="inline">
        {Object.entries(line.attrs).map(([key, value]) => (
          <Fact key={key} label={key}>
            {valueText(value)}
          </Fact>
        ))}
      </Facts>
      {window.isSecureContext && (
        <Button
          size="small"
          onClick={() => {
            void copyLine(line);
          }}
        >
          Copy as JSON
        </Button>
      )}
    </div>
  );
}

function LogLine({
  line,
  expanded,
  onToggle,
}: {
  line: Line;
  expanded: boolean;
  onToggle: () => void;
}) {
  return (
    <div className={expanded ? "bg-fill" : ""}>
      <button
        type="button"
        aria-expanded={expanded}
        onClick={onToggle}
        className="flex w-full items-center gap-3 px-3 py-1 text-left outline-none hover:bg-fill-hover focus-visible:bg-fill-hover"
      >
        <span className="shrink-0 text-ink-3">
          <PlainTime at={line.time} form="stamp" />
        </span>
        {levels[line.level]}
        <span className="shrink-0 text-ink-2">{line.subsystem}</span>
        <span className="min-w-0 flex-1 truncate text-ink-1">{line.msg}</span>
      </button>
      {expanded && <Attributes line={line} />}
    </div>
  );
}

function DividerWords({ divider }: { divider: Divider }) {
  return divider.reason === "restarted" ? (
    <>
      SlipStream restarted at <PlainTime at={divider.at} form="stamp" />
    </>
  ) : (
    "Some lines were skipped"
  );
}

/**
 * A row of the log. The whole row sets in Mono, the one container that does,
 * since a log line is read down a column like a table of values.
 */
export function LogRowView({
  row,
  expanded,
  onToggle,
}: {
  row: LogRow;
  expanded: boolean;
  onToggle: () => void;
}) {
  return (
    <div className={`${dataFont} text-data`}>
      {row.kind === "line" ? (
        <LogLine line={row} expanded={expanded} onToggle={onToggle} />
      ) : (
        <p className="flex items-center gap-3 px-3 py-2 text-ink-2 before:h-px before:flex-1 before:bg-hairline after:h-px after:flex-1 after:bg-hairline">
          <DividerWords divider={row} />
        </p>
      )}
    </div>
  );
}
