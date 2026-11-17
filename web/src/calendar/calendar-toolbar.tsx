import { ArrowLeft01Icon, ArrowRight01Icon } from "@hugeicons/core-free-icons";

import type { WorksContentType, WorksDatedKind } from "../api/gen/model";
import { addMonths, monthKey, monthWords } from "../format/days";
import { Button } from "../ui/button";
import { IconButton } from "../ui/icon-button";
import { Switch } from "../ui/switch";
import { ViewLinks } from "./calendar-links";
import type { CalendarPlace, CalendarSearch } from "./routed";

const chipLook =
  "inline-flex h-6 shrink-0 items-center rounded-chip px-2 text-chip whitespace-nowrap transition-colors duration-(--duration-control) hover:text-ink-1 aria-pressed:bg-selected aria-pressed:text-ink-1 not-aria-pressed:bg-fill not-aria-pressed:text-ink-2";

/** The dated kinds of the Content Types in view, each once, in the types' order. */
export function kindsIn(types: readonly WorksContentType[]): WorksDatedKind[] {
  const seen = new Map<string, WorksDatedKind>();
  for (const kind of types.flatMap(({ dated }) => dated)) {
    seen.set(kind.key, kind);
  }
  return [...seen.values()];
}

/** A toggle chip per dated kind; one stays pressed, and all pressed is the default. */
function KindChips({
  kinds,
  chosen,
  onChange,
}: {
  kinds: readonly WorksDatedKind[];
  chosen: readonly string[] | undefined;
  onChange: (next: readonly string[] | undefined) => void;
}) {
  const pressed = (key: string) => chosen === undefined || chosen.includes(key);
  return (
    <fieldset
      aria-label="Release dates"
      className="flex min-w-0 gap-1.5 overflow-x-auto [scrollbar-width:none]"
    >
      {kinds.map(({ key, label }) => (
        <button
          key={key}
          type="button"
          aria-pressed={pressed(key)}
          className={chipLook}
          onClick={() => {
            const next = kinds
              .map((kind) => kind.key)
              .filter((each) => (each === key ? !pressed(key) : pressed(each)));
            if (next.length > 0) {
              onChange(next.length === kinds.length ? undefined : next);
            }
          }}
        >
          {label}
        </button>
      ))}
    </fieldset>
  );
}

/** Month moves a month at a time and back to Today's; List pages at its ends. */
function MonthControls({
  place,
  search,
  onSearch,
}: {
  place: CalendarPlace;
  search: CalendarSearch;
  onSearch: (next: CalendarSearch) => void;
}) {
  const go = (months: number) => {
    onSearch({ ...search, month: monthKey(addMonths(place.month, months)) });
  };
  return (
    <div className="flex items-center gap-1">
      <IconButton
        icon={ArrowLeft01Icon}
        label="Previous month"
        onClick={() => {
          go(-1);
        }}
      />
      <h2 className="min-w-36 text-center text-heading text-ink-1">{monthWords(place.month)}</h2>
      <IconButton
        icon={ArrowRight01Icon}
        label="Next month"
        onClick={() => {
          go(1);
        }}
      />
      <Button
        size="small"
        onClick={() => {
          const { month: _month, ...rest } = search;
          onSearch(rest);
        }}
      >
        Today
      </Button>
    </div>
  );
}

export function CalendarToolbar({
  types,
  contentType,
  place,
  search,
  onSearch,
}: {
  types: readonly WorksContentType[];
  contentType: string | undefined;
  place: CalendarPlace;
  search: CalendarSearch;
  onSearch: (next: CalendarSearch) => void;
}) {
  const kinds = kindsIn(
    contentType === undefined ? types : types.filter(({ id }) => id === contentType),
  );
  return (
    <div className="mb-4 flex flex-wrap items-center gap-x-4 gap-y-3">
      <ViewLinks contentType={contentType} search={search} />
      {place.view === "month" && (
        <MonthControls place={place} search={search} onSearch={onSearch} />
      )}
      {kinds.length > 1 && (
        <KindChips
          kinds={kinds}
          chosen={place.kinds}
          onChange={(next) => {
            const { kinds: _kinds, ...rest } = search;
            onSearch(next === undefined ? rest : { ...rest, kinds: next.join(",") });
          }}
        />
      )}
      <Switch
        label="Show unmonitored"
        checked={place.unmonitored}
        onCheckedChange={(on) => {
          const { unmonitored: _unmonitored, ...rest } = search;
          onSearch(on ? { ...rest, unmonitored: true } : rest);
        }}
      />
    </div>
  );
}
