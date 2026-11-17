import type { ReactNode } from "react";

import {
  type IsoDay,
  type Month,
  addDays,
  calendarDayWords,
  monthOf,
  weekdayNames,
} from "../format/days";

const weeks = [0, 1, 2, 3, 4, 5] as const;
const weekdays = [0, 1, 2, 3, 4, 5, 6] as const;

const inMonth = (day: IsoDay, { year, month }: Month) => {
  const of = monthOf(day);
  return of.year === year && of.month === month;
};

function WeekdayHeads({ from }: { from: IsoDay }) {
  return (
    <thead>
      <tr>
        {weekdays.map((offset) => {
          const names = weekdayNames(addDays(from, offset));
          return (
            <th
              key={offset}
              scope="col"
              abbr={names.long}
              className="h-8 text-label font-normal text-ink-3"
            >
              {names.short}
            </th>
          );
        })}
      </tr>
    </thead>
  );
}

function Day({
  day,
  month,
  today,
  children,
}: {
  day: IsoDay;
  month: Month;
  today: IsoDay;
  children: ReactNode;
}) {
  const isToday = day === today;
  return (
    <td
      {...(isToday && { "aria-current": "date" as const })}
      className={`relative h-14 border-t border-hairline p-1 align-top not-first:border-l @min-[640px]:h-32 ${inMonth(day, month) ? "" : "[&>*]:opacity-55"}`}
    >
      <span
        aria-hidden="true"
        className={`inline-flex size-6 items-center justify-center rounded-capsule text-meta tabular-nums ${isToday ? "bg-selected font-semibold text-ink-1 ring-1 ring-glow" : "text-ink-2"}`}
      >
        {Number(day.slice(8))}
      </span>
      <span className="sr-only hidden @min-[640px]:block">{calendarDayWords(day, today)}</span>
      {children}
    </td>
  );
}

/**
 * Six weeks of seven days from `from`, the first day of a week, with
 * `month`'s days in full ink and Today marked. Each day's content is the
 * caller's. Below a 640px container the cells are short, for one mark a day,
 * and the caller's content names the day, so the cell has one name.
 */
export function MonthGrid({
  label,
  from,
  month,
  today,
  renderDay,
}: {
  label: string;
  from: IsoDay;
  month: Month;
  today: IsoDay;
  renderDay: (day: IsoDay) => ReactNode;
}) {
  return (
    <div className="@container">
      <table
        aria-label={label}
        className="w-full table-fixed border-separate border-spacing-0 overflow-hidden rounded-card bg-glass ring-1 ring-hairline ring-inset"
      >
        <WeekdayHeads from={from} />
        <tbody>
          {weeks.map((week) => (
            <tr key={week}>
              {weekdays.map((offset) => {
                const day = addDays(from, week * 7 + offset);
                return (
                  <Day key={day} day={day} month={month} today={today}>
                    {renderDay(day)}
                  </Day>
                );
              })}
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}
