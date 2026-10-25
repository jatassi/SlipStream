import { Alert02Icon, ArrowDown01Icon } from "@hugeicons/core-free-icons";

import { Button } from "../ui/button";
import { Icon } from "../ui/icon";
import { Menu } from "../ui/menu";
import { Segmented } from "../ui/segmented";
import { type Rule, type TargetDraft, ruleProblem } from "./draft";
import type { SectionProps } from "./editor";
import { type Value, attributes } from "./vocab";

type Mode = Rule["mode"];

const modes: readonly { value: Mode; label: string }[] = [
  { value: "any", label: "Any" },
  { value: "only", label: "Only these" },
  { value: "not", label: "Not these" },
];

const picked = (values: readonly Value[], keys: readonly string[]): string =>
  keys.length === 0
    ? "Pick values"
    : values
        .filter((value) => keys.includes(value.key))
        .map((value) => value.name)
        .join(", ");

function ValuePicker({
  label,
  values,
  rule,
  onRule,
}: {
  label: string;
  values: readonly Value[];
  rule: Exclude<Rule, { mode: "any" }>;
  onRule: (rule: Rule) => void;
}) {
  const problem = ruleProblem(rule);
  return (
    <div className="flex flex-col items-start gap-1.5">
      <Menu
        label={`${label} values`}
        trigger={
          <Button
            size="small"
            icon={ArrowDown01Icon}
            aria-label={`${label}: ${picked(values, rule.values)}`}
          >
            <span className="max-w-[40ch] truncate">{picked(values, rule.values)}</span>
          </Button>
        }
        items={values.map((value) => ({
          kind: "toggle" as const,
          label: value.name,
          checked: rule.values.includes(value.key),
          onCheckedChange: (on: boolean) => {
            const keys = values
              .map((known) => known.key)
              .filter((key) => (key === value.key ? on : rule.values.includes(key)));
            onRule({ mode: rule.mode, values: keys });
          },
        }))}
      />
      {problem !== undefined && (
        <p className="flex items-center gap-1.5 text-meta text-fail-ink">
          <Icon icon={Alert02Icon} className="size-3.5 shrink-0" />
          {problem}
        </p>
      )}
    </div>
  );
}

function FilterRow({
  label,
  values,
  rule,
  onRule,
}: {
  label: string;
  values: readonly Value[];
  rule: Rule;
  onRule: (rule: Rule) => void;
}) {
  return (
    <li className="flex flex-col gap-2 px-4 py-3">
      <div className="flex flex-wrap items-center justify-between gap-x-4 gap-y-2">
        <span className="font-semibold text-ink-1">{label}</span>
        <Segmented
          label={label}
          options={modes}
          value={rule.mode}
          onValueChange={(mode) => {
            onRule(
              mode === "any" ? { mode } : { mode, values: rule.mode === "any" ? [] : rule.values },
            );
          }}
        />
      </div>
      {rule.mode !== "any" && (
        <ValuePicker label={label} values={values} rule={rule} onRule={onRule} />
      )}
    </li>
  );
}

export function FiltersSection({ draft, onChange }: SectionProps<TargetDraft>) {
  return (
    <div className="flex flex-col gap-3">
      <p className="text-ink-2">
        A release or file that fails a filter never fills this Target. An unknown fact passes, and
        the file is checked again once it is probed.
      </p>
      <ul
        aria-label="Filters"
        className="divide-y divide-hairline rounded-card bg-glass ring-1 ring-hairline ring-inset"
      >
        {attributes.map(({ key, label, values }) => (
          <FilterRow
            key={key}
            label={label}
            values={values}
            rule={draft.filters[key]}
            onRule={(rule) => {
              onChange({ ...draft, filters: { ...draft.filters, [key]: rule } });
            }}
          />
        ))}
      </ul>
    </div>
  );
}
