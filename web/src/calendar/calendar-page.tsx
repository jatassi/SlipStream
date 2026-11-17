import type { WorksContentType } from "../api/gen/model";
import { monthKey } from "../format/days";
import { Page } from "../shell/page";
import { CalendarTypeLinks } from "./calendar-links";
import { CalendarSentence } from "./calendar-sentence";
import { CalendarToolbar, kindsIn } from "./calendar-toolbar";
import { ListView } from "./list-view";
import { MonthView } from "./month-view";
import type { CalendarSearch } from "./routed";
import { useCalendar } from "./use-calendar";

/**
 * The Library's release dates, of every Content Type or of `contentType`
 * alone, as a month or a list, led by what needs attention.
 */
export function CalendarPage({
  types,
  contentType,
  search,
  onSearch,
}: {
  types: readonly WorksContentType[];
  contentType: string | undefined;
  search: CalendarSearch;
  onSearch: (next: CalendarSearch) => void;
}) {
  const { place, answer } = useCalendar(search, contentType, onSearch);
  const kinds = kindsIn(
    contentType === undefined ? types : types.filter(({ id }) => id === contentType),
  );
  const withKind = (place.kinds ?? kinds).length > 1;
  return (
    <Page
      beside={types.length < 2 ? undefined : <CalendarTypeLinks types={types} search={search} />}
    >
      <CalendarSentence answer={answer} place={place} contentType={contentType} />
      <CalendarToolbar
        kinds={kinds}
        contentType={contentType}
        place={place}
        search={search}
        onSearch={onSearch}
      />
      {place.view === "month" ? (
        <MonthView
          key={monthKey(place.month)}
          answer={answer}
          place={place}
          types={types}
          withKind={withKind}
        />
      ) : (
        <ListView
          answer={answer}
          place={place}
          types={types}
          withKind={withKind}
          search={search}
          onSearch={onSearch}
        />
      )}
    </Page>
  );
}
