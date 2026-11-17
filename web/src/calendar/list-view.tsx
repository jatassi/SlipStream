import type {
  CalendarAnswer,
  CalendarRow as CalendarRowWire,
  WorksContentType,
} from "../api/gen/model";
import { type IsoDay, addDays, calendarDayWords, isoDay } from "../format/days";
import { Button } from "../ui/button";
import { List } from "../ui/list";
import { CalendarRow, rowKey } from "./calendar-row";
import { type CalendarPlace, type CalendarSearch, listDays } from "./routed";

function DaySection({
  day,
  rows,
  today,
  types,
}: {
  day: IsoDay;
  rows: readonly CalendarRowWire[];
  today: boolean;
  types: readonly WorksContentType[];
}) {
  return (
    <section
      aria-label={calendarDayWords(day)}
      {...(today && { "aria-current": "date" as const })}
      className="flex w-full flex-col gap-2"
    >
      <h2 className="flex items-center gap-2 text-heading text-ink-1">
        {calendarDayWords(day)}
        {today && <span className="text-label text-glow">Today</span>}
      </h2>
      <List label={calendarDayWords(day)}>
        {rows.map((row) => (
          <CalendarRow key={rowKey(row)} row={row} types={types} />
        ))}
      </List>
    </section>
  );
}

/** Six weeks of days that have rows, from the list's first day, paged six weeks at a time. */
export function ListView({
  answer,
  place,
  types,
  search,
  onSearch,
}: {
  answer: CalendarAnswer;
  place: CalendarPlace;
  types: readonly WorksContentType[];
  search: CalendarSearch;
  onSearch: (next: CalendarSearch) => void;
}) {
  const page = (days: number) => {
    onSearch({ ...search, from: addDays(place.from, days) });
  };
  return (
    <div className="flex flex-col items-start gap-6">
      <Button
        size="small"
        onClick={() => {
          page(-listDays);
        }}
      >
        Show earlier
      </Button>
      {answer.days.map(({ day, rows }) => (
        <DaySection
          key={day}
          day={isoDay(day)}
          rows={rows}
          today={day === answer.today}
          types={types}
        />
      ))}
      <Button
        size="small"
        onClick={() => {
          page(listDays);
        }}
      >
        Show later
      </Button>
    </div>
  );
}
