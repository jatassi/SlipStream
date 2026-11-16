import type { WorksContentType, WorksPresetCounts } from "../../api/gen/model";
import { Select } from "../../ui/select";
import { Switch } from "../../ui/switch";
import { note } from "./note";
import type { AddChoices } from "./choices";
import { countOf } from "./sentence";

const label = "Search after adding";

/**
 * How many Items an add with these choices searches, from the preview's
 * counts. A type of one Item has no counts and searches its Item while
 * monitored. Undefined while the preview can't count, as for a Work already
 * in the library.
 */
export function searchedItems(
  type: WorksContentType,
  counts: WorksPresetCounts | null,
  choices: AddChoices,
): number | undefined {
  if (choices.search === "none") {
    return 0;
  }
  if (type.addSearch.kind === "switch") {
    return choices.preset === "none" ? 0 : 1;
  }
  return counts === null
    ? undefined
    : (countOf(counts, choices.preset).searches.find(({ scope }) => scope === choices.search)
        ?.items ?? 0);
}

/** The note under the choice when the search and the monitoring don't overlap. */
export function mismatchWords(type: WorksContentType, choices: AddChoices): string {
  const scope = type.addSearch.choices.find(({ value }) => value === choices.search);
  const where =
    choices.search === "monitored" || scope === undefined
      ? "Nothing"
      : `Nothing in the ${scope.label.toLowerCase()}`;
  return `${where} is monitored, so SlipStream won't search after adding.`;
}

/** "Search after adding": a switch for a type of one Item, else its scopes, with the mismatch note. */
export function SearchChoice({
  type,
  counts,
  choices,
  choose,
}: {
  type: WorksContentType;
  counts: WorksPresetCounts | null;
  choices: AddChoices;
  choose: (next: Partial<AddChoices>) => void;
}) {
  const onChange = (search: AddChoices["search"]) => {
    choose({ search });
  };
  const { kind, choices: scopes } = type.addSearch;
  return (
    <div className="flex flex-col gap-1">
      {kind === "switch" ? (
        <Switch
          label={label}
          checked={choices.search !== "none"}
          onCheckedChange={(on) => {
            onChange(on ? "monitored" : "none");
          }}
        />
      ) : (
        <Select
          label={label}
          options={scopes.map(({ value, label: scope }) => ({ value, label: scope }))}
          value={choices.search}
          onValueChange={onChange}
        />
      )}
      {choices.search !== "none" && searchedItems(type, counts, choices) === 0 && (
        <p className={note}>{mismatchWords(type, choices)}</p>
      )}
    </div>
  );
}
