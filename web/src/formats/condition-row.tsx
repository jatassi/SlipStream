import { Delete02Icon } from "@hugeicons/core-free-icons";

import { DraftField } from "../targets/field";
import { Checkbox } from "../ui/checkbox";
import { Details } from "../ui/details";
import { IconButton } from "../ui/icon-button";
import { Select } from "../ui/select";
import { clauseOf } from "./condition-words";
import { type ConditionDraft, type Kind, type Problems, blankValue, valueWords } from "./draft";
import { ValueField } from "./value-field";

const metaLook = "text-meta text-ink-3 in-data-[density=touch]:text-body-touch";

function Flag({
  label,
  meaning,
  checked,
  onChecked,
}: {
  label: string;
  meaning: string;
  checked: boolean;
  onChecked: (checked: boolean) => void;
}) {
  return (
    <div className="flex flex-col">
      <Checkbox label={label} checked={checked} onCheckedChange={onChecked} />
      <p className={`pl-6.5 ${metaLook}`}>{meaning}</p>
    </div>
  );
}

const sentenceOf = (clause: string) => `${clause.charAt(0).toUpperCase()}${clause.slice(1)}.`;

function ConditionDetails({
  condition,
  onChange,
}: {
  condition: ConditionDraft;
  onChange: (next: ConditionDraft) => void;
}) {
  const clause = clauseOf(condition);
  return (
    <Details
      summary={
        clause === undefined ? `${condition.kind.words} with no value yet.` : sentenceOf(clause)
      }
      fields
    >
      <div className="flex flex-col gap-3 pt-2">
        <div className="max-w-[400px]">
          <DraftField
            label="Name"
            value={condition.name}
            placeholder={valueWords(condition)}
            description="Left empty, the condition takes its value as its name."
            onChange={(name) => {
              onChange({ ...condition, name });
            }}
          />
        </div>
        <Flag
          label="Negate"
          meaning="The condition matches when its test fails."
          checked={condition.negate}
          onChecked={(negate) => {
            onChange({ ...condition, negate });
          }}
        />
        <Flag
          label="Required"
          meaning="The format matches only when this condition does. The conditions of its kind that aren't required then don't count."
          checked={condition.required}
          onChecked={(required) => {
            onChange({ ...condition, required });
          }}
        />
      </div>
    </Details>
  );
}

function KindField({
  condition,
  kinds,
  problem,
  onChange,
}: {
  condition: ConditionDraft;
  kinds: readonly Kind[];
  problem: string | undefined;
  onChange: (next: ConditionDraft) => void;
}) {
  return (
    <div className="w-full min-w-0 sm:w-52">
      <Select
        label="Kind"
        options={kinds.map((kind) => ({ value: kind.kind, label: kind.words }))}
        value={condition.kind.kind}
        onValueChange={(name) => {
          const kind = kinds.find((offered) => offered.kind === name) ?? condition.kind;
          onChange({ ...condition, kind, value: blankValue(kind) });
        }}
      />
      {problem !== undefined && <p className="mt-1.5 text-meta text-fail-ink">{problem}</p>}
    </div>
  );
}

export function ConditionRow({
  condition,
  index,
  kinds,
  problems,
  onChange,
  onRemove,
}: {
  condition: ConditionDraft;
  index: number;
  kinds: readonly Kind[];
  problems: Problems;
  onChange: (next: ConditionDraft) => void;
  onRemove: () => void;
}) {
  const at = `conditions[${String(index)}]`;
  const named = condition.name.trim() || valueWords(condition) || `condition ${String(index + 1)}`;
  return (
    <li aria-label={`Condition ${String(index + 1)}`} className="flex flex-col gap-3 px-4 py-3">
      <div className="flex flex-wrap items-start gap-3">
        <KindField
          condition={condition}
          kinds={kinds}
          problem={problems[`${at}.kind`]}
          onChange={onChange}
        />
        <div className="min-w-0 flex-[1_1_16rem]">
          <ValueField
            kind={condition.kind}
            value={condition.value}
            at={at}
            problems={problems}
            onValue={(value) => {
              onChange({ ...condition, value });
            }}
          />
        </div>
        <div className="pt-6.5">
          <IconButton icon={Delete02Icon} label={`Remove ${named}`} onClick={onRemove} />
        </div>
      </div>
      <ConditionDetails condition={condition} onChange={onChange} />
    </li>
  );
}
