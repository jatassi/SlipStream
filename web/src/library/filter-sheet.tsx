import { FilterHorizontalIcon } from "@hugeicons/core-free-icons";

import { Button } from "../ui/button";
import { Checkbox } from "../ui/checkbox";
import { Popover } from "../ui/popover";
import { Sheet } from "../ui/sheet";
import { useShell } from "../ui/shell-context";
import { type FilterGroup, type RangeGroup, type ValuesGroup, optionsOf } from "./filters";
import type { LibraryWork } from "./rows";
import type { ArrangeContext, Filters, Pick } from "./view";

type EditorProps = {
  readonly groups: readonly FilterGroup[];
  /** The Works of the Library's types, whose values the groups offer. */
  readonly works: readonly LibraryWork[];
  readonly ctx: ArrangeContext;
  readonly filters: Filters;
  readonly onFilters: (filters: Filters) => void;
};

function withPick(filters: Filters, group: FilterGroup, pick: Pick | undefined): Filters {
  const { [group.key]: _, ...rest } = filters;
  return pick === undefined ? rest : { ...rest, [group.key]: pick };
}

function Values({ group, works, ctx, filters, onFilters }: EditorProps & { group: ValuesGroup }) {
  const options = optionsOf(group, works, ctx);
  const pick = filters[group.key];
  const picked = pick?.kind === "values" ? pick.values : [];
  if (options.length === 0) {
    return null;
  }
  return (
    <fieldset className="flex flex-col">
      <legend className="mb-1 text-heading text-ink-1">{group.label}</legend>
      {options.map(({ value, label, count }) => (
        <Checkbox
          key={value}
          label={`${label} (${count})`}
          checked={picked.includes(value)}
          onCheckedChange={(on) => {
            const values = on ? [...picked, value] : picked.filter((each) => each !== value);
            onFilters(
              withPick(
                filters,
                group,
                values.length === 0 ? undefined : { kind: "values", values },
              ),
            );
          }}
        />
      ))}
    </fieldset>
  );
}

const bound = (text: string) =>
  text.trim() === "" || !Number.isFinite(Number(text)) ? undefined : Number(text);

function Range({ group, works, filters, onFilters }: EditorProps & { group: RangeGroup }) {
  if (!works.some((work) => group.numberOf(work) !== undefined)) {
    return null;
  }
  const pick = filters[group.key];
  const range = pick?.kind === "range" ? pick : { min: undefined, max: undefined };
  const set = (min: number | undefined, max: number | undefined) => {
    onFilters(
      withPick(
        filters,
        group,
        min === undefined && max === undefined ? undefined : { kind: "range", min, max },
      ),
    );
  };
  const unit = group.unit === "" ? "" : `, in ${group.unit}`;
  return (
    <fieldset className="flex flex-col gap-2">
      <legend className="mb-1 text-heading text-ink-1">{group.label}</legend>
      <div className="flex gap-3">
        {(["min", "max"] as const).map((end) => (
          <label
            key={end}
            className="flex min-w-0 flex-1 flex-col gap-1.5 text-body font-semibold text-ink-2"
          >
            {`${end === "min" ? "From" : "To"}${unit}`}
            <input
              type="number"
              inputMode="numeric"
              value={range[end] ?? ""}
              onChange={(event) => {
                const value = bound(event.currentTarget.value);
                set(end === "min" ? value : range.min, end === "max" ? value : range.max);
              }}
              className="h-(--control-height) w-full min-w-0 rounded-item bg-well px-3 text-body font-normal text-ink-1 tabular-nums ring-1 ring-hairline ring-inset in-data-[density=touch]:text-input-touch"
            />
          </label>
        ))}
      </div>
    </fieldset>
  );
}

function FilterEditor(props: EditorProps) {
  const { groups, works, filters, onFilters } = props;
  return (
    <div className="flex flex-col gap-5">
      {groups.map((group) => {
        if (group.kind === "values") {
          return <Values key={group.key} {...props} group={group} />;
        }
        if (group.kind === "range") {
          return <Range key={group.key} {...props} group={group} />;
        }
        const count = works.filter((work) => group.has(work)).length;
        return (
          <Checkbox
            key={group.key}
            label={`${group.label} (${count})`}
            checked={filters[group.key] !== undefined}
            onCheckedChange={(on) => {
              onFilters(withPick(filters, group, on ? { kind: "flag" } : undefined));
            }}
          />
        );
      })}
      {Object.keys(filters).length > 0 && (
        <Button
          look="ghost"
          className="self-start"
          onClick={() => {
            onFilters({});
          }}
        >
          Clear filters
        </Button>
      )}
    </div>
  );
}

/** The Filter button: a popover of every group in the desktop shell, a sheet on phone. */
export function FilterButton(props: EditorProps) {
  const active = Object.keys(props.filters).length;
  const trigger = (
    <Button icon={FilterHorizontalIcon}>{active === 0 ? "Filter" : `Filter (${active})`}</Button>
  );
  if (useShell() === "phone") {
    return (
      <Sheet title="Filters" trigger={trigger}>
        <div className="pb-2">
          <FilterEditor {...props} />
        </div>
      </Sheet>
    );
  }
  return (
    <Popover label="Filters" trigger={trigger}>
      <div className="max-h-[70vh] overflow-y-auto">
        <FilterEditor {...props} />
      </div>
    </Popover>
  );
}
