import type { ReactElement } from "react";

import { Button } from "../../../../ui/button";
import { Chip } from "../../../../ui/chip";
import { dataFont } from "../../../../ui/data";
import { SignalChip } from "../../../../ui/signal-chip";
import { toast } from "../../../../ui/toast";
import type { Level, LogRow } from "./buffer";

type Line = Extract<LogRow, { kind: "line" }>;
type Divider = Extract<LogRow, { kind: "divider" }>;

const pad = (value: number, width = 2) => String(value).padStart(width, "0");

const secondsOf = (date: Date) =>
  `${pad(date.getHours())}:${pad(date.getMinutes())}:${pad(date.getSeconds())}`;

/** A record's time in the browser's zone, as HH:MM:SS.mmm. */
export const clockOf = (iso: string): string => {
  const date = new Date(iso);
  return `${secondsOf(date)}.${pad(date.getMilliseconds(), 3)}`;
};

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
      <dl className="grid grid-cols-[max-content_minmax(0,1fr)] gap-x-4 gap-y-1">
        {Object.entries(line.attrs).map(([key, value]) => (
          <div key={key} className="contents">
            <dt className="text-ink-3">{key}</dt>
            <dd className="break-words text-ink-1">{valueText(value)}</dd>
          </div>
        ))}
      </dl>
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
        <time dateTime={line.time} className="shrink-0 text-ink-3">
          {clockOf(line.time)}
        </time>
        {levels[line.level]}
        <span className="shrink-0 text-ink-2">{line.subsystem}</span>
        <span className="min-w-0 flex-1 truncate text-ink-1">{line.msg}</span>
      </button>
      {expanded && <Attributes line={line} />}
    </div>
  );
}

const dividerWords = (divider: Divider) =>
  divider.reason === "restarted"
    ? `SlipStream restarted at ${secondsOf(new Date(divider.at))}`
    : "Some lines were skipped";

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
          {dividerWords(row)}
        </p>
      )}
    </div>
  );
}
