import type {
  CalendarAnswer,
  CalendarRow as CalendarRowWire,
  WorksContentType,
} from "../api/gen/model";
import { type IsoDay, addDays, calendarDayWords, isoDay, spanWords } from "../format/days";
import { Button } from "../ui/button";
import { List } from "../ui/list";
import { CalendarRow, rowKey } from "./calendar-row";
import { listEmptyWords } from "./calendar-words";
import { type CalendarPlace, type CalendarSearch, listDays } from "./routed";

function DaySection({
  day,
  rows,
  today,
  types,
  withKind,
}: {
  day: IsoDay;
  rows: readonly CalendarRowWire[];
  today: IsoDay;
  types: readonly WorksContentType[];
  withKind: boolean;
}) {
  const words = calendarDayWords(day, today);
  const isToday = day === today;
  return (
    <section
      aria-label={words}
      {...(isToday && { "aria-current": "date" as const })}
      className="flex w-full flex-col gap-2"
    >
      <h3 className="flex items-center gap-2 text-heading text-ink-1">
        {words}
        {isToday && <span className="text-label text-glow">Today</span>}
      </h3>
      <List label={words}>
        {rows.map((row) => (
          <CalendarRow key={rowKey(row)} row={row} types={types} withKind={withKind} />
        ))}
      </List>
    </section>
  );
}

/** The span's days that have rows, or the words saying it has none. */
function Days({
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
  if (answer.days.length === 0) {
    return <p className="text-ink-2">{listEmptyWords(place)}</p>;
  }
  const today = isoDay(answer.today);
  return answer.days.map(({ day, rows }) => (
    <DaySection
      key={day}
      day={isoDay(day)}
      rows={rows}
      today={today}
      types={types}
      withKind={withKind}
    />
  ));
}

/**
 * Six weeks of days that have rows, from the list's first day, under the
 * span's range and paged six weeks at a time. An empty span says so between
 * its paging buttons.
 */
export function ListView({
  answer,
  place,
  types,
  withKind,
  search,
  onSearch,
}: {
  answer: CalendarAnswer;
  place: CalendarPlace;
  types: readonly WorksContentType[];
  withKind: boolean;
  search: CalendarSearch;
  onSearch: (next: CalendarSearch) => void;
}) {
  const page = (days: number) => {
    onSearch({ ...search, from: addDays(place.from, days) });
  };
  return (
    <div className="flex flex-col items-start gap-6 @container">
      <h2 className="text-heading text-ink-1">{spanWords(place.from, addDays(place.to, -1))}</h2>
      <Button
        size="small"
        onClick={() => {
          page(-listDays);
        }}
      >
        Show earlier
      </Button>
      <Days answer={answer} place={place} types={types} withKind={withKind} />
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
