import { Link } from "@tanstack/react-router";

import type { CalendarRow as Row, FilesState, WorksContentType } from "../api/gen/model";
import { type IsoDay, calendarDayWords } from "../format/days";
import { Icon } from "../ui/icon";
import { List } from "../ui/list";
import { Popover } from "../ui/popover";
import { Artwork } from "../ui/artwork";
import { notMonitored } from "../ui/state-mark";
import { type StatusMark, type Tone, stateOrder, statusMarks } from "../ui/state-marks";
import { CalendarRow, rowKey, rowWords } from "./calendar-row";

const toneInks: { readonly [T in Tone]: string } = {
  fail: "text-fail-ink",
  warn: "text-warn",
  quiet: "text-ink-3",
  neutral: "text-ink-2",
  live: "text-glow",
  met: "text-ink-1",
};

/** A mark's icon alone, its word for a screen reader. */
function MarkIcon({ mark }: { mark: StatusMark }) {
  return (
    <span className={`inline-flex shrink-0 ${toneInks[mark.tone]}`}>
      <Icon icon={mark.icon} className="size-3.5" />
      <span className="sr-only">{mark.word}</span>
    </span>
  );
}

/** The state that needs attention first among the rows, in the one status order. */
function firstState(rows: readonly Row[]): FilesState | undefined {
  const states = new Set(rows.map(({ state }) => state));
  return stateOrder.find((state) => states.has(state));
}

/**
 * A row as a grid cell fits it: poster, label and words over two lines, and
 * the icons of its state and Not monitored. Its title holds the whole words.
 */
function CellRow({ row, withKind }: { row: Row; withKind: boolean }) {
  const words = rowWords(row, withKind);
  return (
    <li>
      <Link
        to="/works/$workId"
        params={{ workId: row.work.id }}
        title={words === "" ? row.work.label : `${row.work.label} · ${words}`}
        className="flex items-center gap-1.5 rounded-item p-0.5 transition-colors duration-350 ease-standard hover:bg-(--row-wash) focus-visible:bg-(--row-wash)"
      >
        <Artwork use="poster" size="option" url={row.work.posterUrl ?? undefined} />
        <span className="flex min-w-0 flex-1 flex-col text-meta">
          <span className="truncate text-ink-1">{row.work.label}</span>
          {words !== "" && <span className="line-clamp-2 text-ink-3">{words}</span>}
        </span>
        {row.state !== null && <MarkIcon mark={statusMarks[row.state]} />}
        {!row.monitored && <MarkIcon mark={notMonitored} />}
      </Link>
    </li>
  );
}

const fits = 3;

/** The whole day in a popover, for a day with more rows than its cell fits. */
function MoreRows({
  day,
  today,
  rows,
  more,
  types,
  withKind,
}: {
  day: IsoDay;
  today: IsoDay;
  rows: readonly Row[];
  more: number;
  types: readonly WorksContentType[];
  withKind: boolean;
}) {
  const words = calendarDayWords(day, today);
  return (
    <Popover
      label={words}
      width={400}
      trigger={
        <button type="button" className="rounded-item px-1 text-meta text-ink-2 hover:text-ink-1">
          +{more} more
        </button>
      }
    >
      <h3 className="mb-3 text-heading text-ink-1">{words}</h3>
      <List label={words}>
        {rows.map((row) => (
          <CalendarRow key={rowKey(row)} row={row} types={types} withKind={withKind} />
        ))}
      </List>
    </Popover>
  );
}

/** A narrow grid's day: a button with the day's first state, which picks the day. */
function DayButton({
  day,
  today,
  rows,
  picked,
  onPick,
}: {
  day: IsoDay;
  today: IsoDay;
  rows: readonly Row[];
  picked: boolean;
  onPick: (day: IsoDay) => void;
}) {
  const state = firstState(rows);
  return (
    <button
      type="button"
      aria-pressed={picked}
      className="absolute inset-0 flex items-end justify-center pb-1.5 aria-pressed:bg-selected @min-[640px]:hidden"
      onClick={() => {
        onPick(day);
      }}
    >
      <span className="sr-only">{calendarDayWords(day, today)}</span>
      {state === undefined ? (
        rows.length > 0 && <span className="size-1.5 rounded-capsule bg-ink-2" />
      ) : (
        <MarkIcon mark={statusMarks[state]} />
      )}
    </button>
  );
}

/**
 * One day of the month. In a wide grid it lists its rows until the cell is
 * full, then "+3 more" opens the whole day. In a narrow one it is a button
 * with the day's first state, which shows the day's rows under the grid.
 */
export function DayCell({
  day,
  today,
  rows,
  types,
  withKind,
  picked,
  onPick,
}: {
  day: IsoDay;
  today: IsoDay;
  rows: readonly Row[];
  types: readonly WorksContentType[];
  withKind: boolean;
  picked: boolean;
  onPick: (day: IsoDay) => void;
}) {
  const shown = rows.length > fits ? rows.slice(0, fits - 1) : rows;
  return (
    <>
      {rows.length > 0 && (
        <ul className="mt-1 hidden flex-col gap-1 @min-[640px]:flex">
          {shown.map((row) => (
            <CellRow key={rowKey(row)} row={row} withKind={withKind} />
          ))}
          {shown.length < rows.length && (
            <li>
              <MoreRows
                day={day}
                today={today}
                rows={rows}
                more={rows.length - shown.length}
                types={types}
                withKind={withKind}
              />
            </li>
          )}
        </ul>
      )}
      <DayButton day={day} today={today} rows={rows} picked={picked} onPick={onPick} />
    </>
  );
}
