import { MoreHorizontalIcon, Sorting01Icon } from "@hugeicons/core-free-icons";

import { Button } from "../ui/button";
import { IconButton } from "../ui/icon-button";
import { Menu, type MenuEntry } from "../ui/menu";
import { SearchField } from "../ui/search-field";
import type { Scope } from "./arrange";
import type { ColumnDef } from "./columns";
import { FilterButton } from "./filter-sheet";
import type { LibraryWork } from "./rows";
import type { ArrangeContext, LibraryView, PosterSize } from "./view";

const sizeLabels: { readonly [S in PosterSize]: string } = {
  small: "Small posters",
  medium: "Medium posters",
  large: "Large posters",
};

export type ToolbarProps = {
  readonly scope: Scope;
  readonly view: LibraryView;
  readonly onView: (change: Partial<LibraryView>) => void;
  readonly works: readonly LibraryWork[];
  readonly ctx: ArrangeContext;
  readonly columns: readonly ColumnDef[];
  readonly noun: string;
  readonly onSelect: () => void;
  /** Refreshes every Work of the one type this Library shows. */
  readonly onRefreshAll: (() => void) | undefined;
};

function SortMenu({ scope, view, onView }: ToolbarProps) {
  const current = scope.sorts.find(({ key }) => key === view.sort) ?? scope.sorts[0];
  const items: MenuEntry[] = [
    ...scope.sorts.map(({ key, label }) => ({
      kind: "toggle" as const,
      label,
      checked: key === current?.key,
      onCheckedChange: () => {
        onView({ sort: key });
      },
    })),
    "separator",
    {
      kind: "toggle",
      label: "Reverse",
      checked: view.reverse,
      onCheckedChange: (reverse) => {
        onView({ reverse });
      },
    },
  ];
  return (
    <Menu
      label="Sort"
      trigger={
        <Button icon={Sorting01Icon}>
          {`Sort: ${current?.label ?? ""}${view.reverse ? ", reversed" : ""}`}
        </Button>
      }
      items={items}
    />
  );
}

function shownColumns(
  view: LibraryView,
  columns: readonly ColumnDef[],
  key: ColumnDef["key"],
  on: boolean,
) {
  return columns
    .map((column) => column.key)
    .filter((each) => (each === key ? on : view.columns.includes(each)));
}

function viewItems({ view, onView, columns }: ToolbarProps): MenuEntry[] {
  const layouts: MenuEntry[] = (["grid", "table"] as const).map((layout) => ({
    kind: "toggle",
    label: layout === "grid" ? "Grid" : "Table",
    checked: view.layout === layout,
    onCheckedChange: () => {
      onView({ layout });
    },
  }));
  const details: MenuEntry[] =
    view.layout === "grid"
      ? (["small", "medium", "large"] as const).map((size) => ({
          kind: "toggle",
          label: sizeLabels[size],
          checked: view.size === size,
          onCheckedChange: () => {
            onView({ size });
          },
        }))
      : columns.map(({ key, header }) => ({
          kind: "toggle",
          label: header,
          checked: view.columns.includes(key),
          onCheckedChange: (on) => {
            onView({ columns: shownColumns(view, columns, key, on) });
          },
        }));
  return [...layouts, "separator", ...details];
}

function OptionsMenu(props: ToolbarProps) {
  const { onSelect, onRefreshAll } = props;
  return (
    <Menu
      label="Library options"
      trigger={<IconButton icon={MoreHorizontalIcon} label="Library options" />}
      items={[
        ...viewItems(props),
        "separator",
        { label: "Select", onSelect },
        ...(onRefreshAll === undefined ? [] : [{ label: "Refresh all", onSelect: onRefreshAll }]),
      ]}
    />
  );
}

/** The Library's search box, Filter, Sort and its "…" menu. */
export function Toolbar(props: ToolbarProps) {
  const { scope, view, onView, works, ctx, noun } = props;
  return (
    <div className="mb-4 flex flex-wrap items-center gap-2">
      <SearchField
        label={`Search ${noun}`}
        placeholder="Search by any title"
        value={view.q}
        onValueChange={(q) => {
          onView({ q });
        }}
        className="w-72 max-w-full min-w-0 grow sm:grow-0"
      />
      <FilterButton
        groups={scope.groups}
        works={works}
        ctx={ctx}
        filters={view.filters}
        onFilters={(filters) => {
          onView({ filters });
        }}
      />
      <SortMenu {...props} />
      <OptionsMenu {...props} />
    </div>
  );
}
