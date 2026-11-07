import { Search01Icon } from "@hugeicons/core-free-icons";
import { type KeyboardEvent, useId, useState } from "react";

import type { IndexersCatalogEntry, IndexersIndexerCatalog } from "../../api/gen/model";
import { Icon } from "../../ui/icon";
import { type Move, moveIndex } from "../../ui/move-index";
import { VirtualListbox, optionId } from "../../ui/virtual-listbox";
import { type Filter, anyLanguage, filtered, languagesOf } from "./picker-filter";
import { NotYet } from "./not-yet";
import { EntryRow, Filters } from "./picker-parts";

const keys: Readonly<Record<string, Move>> = {
  ArrowDown: "next",
  ArrowUp: "prev",
  Home: "first",
  End: "last",
  PageDown: "pageDown",
  PageUp: "pageUp",
};

const page = 8;

/** The search field's keys: arrows move through the rows, Enter picks the active one. */
function usePicking(
  shown: readonly IndexersCatalogEntry[],
  onPick: (entry: IndexersCatalogEntry) => void,
) {
  const [active, setActive] = useState(-1);
  const onKeyDown = (event: KeyboardEvent<HTMLInputElement>) => {
    const move = keys[event.key];
    if (move !== undefined) {
      event.preventDefault();
      setActive((from) => moveIndex(from, move, shown.length, page));
      return;
    }
    const entry = shown[active];
    if (event.key === "Enter" && entry !== undefined) {
      event.preventDefault();
      onPick(entry);
    }
  };
  return { active, setActive, onKeyDown };
}

function Rows({
  id,
  shown,
  active,
  query,
  onPick,
}: {
  id: string;
  shown: readonly IndexersCatalogEntry[];
  active: number;
  query: string;
  onPick: (entry: IndexersCatalogEntry) => void;
}) {
  if (shown.length === 0) {
    return (
      <p className="p-5 text-ink-2">
        {query.trim() === ""
          ? "No definition matches these filters."
          : `No definition matches ${query.trim()}.`}
      </p>
    );
  }
  return (
    <VirtualListbox
      id={id}
      label="Indexer catalog"
      rows={shown}
      rowKey={(entry) => `${entry.origin}:${entry.id}`}
      estimate={60}
      active={active}
      onChoose={(index) => {
        const entry = shown[index];
        if (entry !== undefined) {
          onPick(entry);
        }
      }}
      renderRow={(entry, isActive) => <EntryRow entry={entry} active={isActive} />}
    />
  );
}

function SearchInput({
  listId,
  listed,
  active,
  query,
  onQuery,
  onKeyDown,
}: {
  listId: string;
  listed: boolean;
  active: number;
  query: string;
  onQuery: (query: string) => void;
  onKeyDown: (event: KeyboardEvent<HTMLInputElement>) => void;
}) {
  return (
    <div className="relative flex items-center">
      <Icon icon={Search01Icon} className="pointer-events-none absolute left-3 size-4 text-ink-3" />
      <input
        type="search"
        role="combobox"
        aria-label="Search the Indexer catalog"
        aria-autocomplete="list"
        aria-expanded={listed}
        aria-controls={listed ? listId : undefined}
        aria-activedescendant={listed && active >= 0 ? optionId(listId, active) : undefined}
        placeholder="Search by name"
        autoComplete="off"
        spellCheck={false}
        value={query}
        onChange={(event) => {
          onQuery(event.target.value);
        }}
        onKeyDown={onKeyDown}
        className="h-(--control-height) w-full min-w-0 rounded-capsule bg-fill pr-3.5 pl-9 text-body text-ink-1 ring-1 ring-hairline ring-inset placeholder:text-ink-3 in-data-[density=touch]:text-input-touch"
      />
    </div>
  );
}

/**
 * The Indexer catalog, searchable by name and narrowed by privacy and
 * language. Focus stays in the search field, whose arrow keys move through
 * the rows; Enter or a click picks one.
 */
export function CatalogPicker({
  catalog,
  onPick,
}: {
  catalog: IndexersIndexerCatalog;
  onPick: (entry: IndexersCatalogEntry) => void;
}) {
  const listId = useId();
  const [filter, setFilter] = useState<Filter>({
    query: "",
    privacy: "all",
    language: anyLanguage,
  });
  const { shown, counts } = filtered(catalog.entries, filter);
  const { active, setActive, onKeyDown } = usePicking(shown, onPick);
  if (catalog.notice !== null) {
    return <NotYet notice={catalog.notice} />;
  }
  const change = (next: Filter) => {
    setFilter(next);
    setActive(-1);
  };
  return (
    <div className="flex flex-col gap-3">
      <SearchInput
        listId={listId}
        listed={shown.length > 0}
        active={active}
        query={filter.query}
        onQuery={(query) => {
          change({ ...filter, query });
        }}
        onKeyDown={onKeyDown}
      />
      <Filters
        filter={filter}
        counts={counts}
        languages={languagesOf(catalog.entries)}
        onChange={change}
      />
      <div className="flex h-[min(60vh,560px)] flex-col overflow-hidden rounded-card bg-glass ring-1 ring-hairline ring-inset">
        <Rows id={listId} shown={shown} active={active} query={filter.query} onPick={onPick} />
      </div>
    </div>
  );
}
