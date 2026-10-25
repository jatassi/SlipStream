import { Data } from "../ui/data";
import { Section } from "../ui/section";
import { SignalRow } from "../ui/signal-row";
import type { Row } from "./model";
import type { Signal } from "./summary";
import type { Names } from "./names";

const subjectOf = (signal: Signal, names: Names) => {
  const [only] = signal.rows;
  return signal.rows.length === 1 && only !== undefined
    ? (names.work(only.workId) ?? only.name)
    : `${signal.rows.length} torrents`;
};

function Held({ row, names }: { row: Row; names: Names }) {
  return (
    <li className="flex min-w-0 flex-col gap-0.5 py-1.5">
      <span className="truncate font-semibold">{names.work(row.workId) ?? row.name}</span>
      <span className="truncate opacity-80">
        <Data>{row.name}</Data> · {names.node(row.nodeId)}
      </span>
      {row.fault && <span>{row.fault.sentence}</span>}
    </li>
  );
}

export function Signals({ signals, names }: { signals: readonly Signal[]; names: Names }) {
  return (
    <Section title="Needs you" count={signals.reduce((sum, signal) => sum + signal.rows.length, 0)}>
      <div className="flex flex-col gap-2">
        {signals.map((signal) => (
          <div key={signal.anchor} id={signal.anchor} className="scroll-mt-20">
            <SignalRow
              signal={signal.signal}
              chip={signal.chip}
              subject={subjectOf(signal, names)}
              verdict={signal.verdict}
              time=""
            >
              <ul className="flex flex-col text-(length:--body-size) leading-(--body-line-height)">
                {signal.rows.map((row) => (
                  <Held key={row.id} row={row} names={names} />
                ))}
              </ul>
            </SignalRow>
          </div>
        ))}
      </div>
    </Section>
  );
}
