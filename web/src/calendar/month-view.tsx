import { useState } from "react";

import type {
  CalendarAnswer,
  CalendarRow as CalendarRowWire,
  WorksContentType,
} from "../api/gen/model";
import {
  type IsoDay,
  calendarDayWords,
  isoDay,
  monthKey,
  monthOf,
  monthWords,
} from "../format/days";
import { List } from "../ui/list";
import { MonthGrid } from "../ui/month-grid";
import { CalendarRow, rowKey } from "./calendar-row";
import { DayCell } from "./day-cell";
import type { CalendarPlace } from "./routed";

const sameMonth = (day: IsoDay, { month }: CalendarPlace) => {
  const of = monthOf(day);
  return of.year === month.year && of.month === month.month;
};

/** A narrow month's picked day, its rows under the grid. */
function PickedDay({
  words,
  rows,
  types,
  withKind,
}: {
  words: string;
  rows: readonly CalendarRowWire[];
  types: readonly WorksContentType[];
  withKind: boolean;
}) {
  return (
    <section aria-label={words} className="mt-4 @min-[640px]:hidden">
      <h2 className="mb-2 text-heading text-ink-1">{words}</h2>
      {rows.length === 0 ? (
        <p className="text-meta text-ink-3">Nothing in your library comes out on this day.</p>
      ) : (
        <List label={words}>
          {rows.map((row) => (
            <CalendarRow key={rowKey(row)} row={row} types={types} withKind={withKind} />
          ))}
        </List>
      )}
    </section>
  );
}

/**
 * The month's six weeks. In a narrow container each day shows one mark, and
 * the picked day's rows list under the grid, Today's until another is picked.
 */
export function MonthView({
  answer,
  place,
  types,
  withKind,
}: {
  answer: CalendarAnswer;
  place: CalendarPlace;
  types: readonly WorksContentType[];
  withKind: boolean;
}) {
  const today = isoDay(answer.today);
  const rowsOf = new Map(answer.days.map(({ day, rows }) => [day, rows]));
  const [chosen, setChosen] = useState<IsoDay>();
  const picked =
    chosen ?? (sameMonth(today, place) ? today : isoDay(`${monthKey(place.month)}-01`));
  return (
    <div className="@container">
      <MonthGrid
        label={monthWords(place.month)}
        from={place.from}
        month={place.month}
        today={today}
        renderDay={(day) => (
          <DayCell
            day={day}
            today={today}
            rows={rowsOf.get(day) ?? []}
            types={types}
            withKind={withKind}
            picked={day === picked}
            onPick={setChosen}
          />
        )}
      />
      <PickedDay
        words={calendarDayWords(picked, today)}
        rows={rowsOf.get(picked) ?? []}
        types={types}
        withKind={withKind}
      />
    </div>
  );
}
