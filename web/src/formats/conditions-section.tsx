import { Add01Icon } from "@hugeicons/core-free-icons";

import type { FormatsSchema } from "../api/gen/model";
import { AlertLine } from "../ui/alert-line";
import { Button } from "../ui/button";
import { SignalChip } from "../ui/signal-chip";
import { ConditionRow } from "./condition-row";
import { matchSentence, slowSentence } from "./condition-words";
import { type FormatDraft, type Kind, type Problems, blankCondition, kindsIn } from "./draft";
import { appWords } from "./paste-words";

export type Slow = { readonly releases: number; readonly limit: number };

function SlowWarning({ slow }: { slow: Slow }) {
  return (
    <output className="flex flex-wrap items-center gap-x-3 gap-y-2 rounded-card bg-warn-card px-4 py-3 text-warn-text ring-1 ring-warn-line ring-inset">
      <SignalChip kind="warn">Slow pattern</SignalChip>
      <span className="min-w-0 flex-[1_1_24ch]">{slowSentence(slow.releases, slow.limit)}</span>
    </output>
  );
}

function ConditionList({
  draft,
  kinds,
  problems,
  onChange,
}: {
  draft: FormatDraft;
  kinds: readonly Kind[];
  problems: Problems;
  onChange: (next: FormatDraft) => void;
}) {
  return (
    <ul
      aria-label="Conditions"
      className="divide-y divide-hairline rounded-card bg-glass ring-1 ring-hairline ring-inset"
    >
      {draft.conditions.map((condition, index) => (
        <ConditionRow
          key={condition.key}
          condition={condition}
          index={index}
          kinds={kinds}
          problems={problems}
          onChange={(next) => {
            onChange({ ...draft, conditions: draft.conditions.with(index, next) });
          }}
          onRemove={() => {
            onChange({ ...draft, conditions: draft.conditions.toSpliced(index, 1) });
          }}
        />
      ))}
    </ul>
  );
}

export function ConditionsSection({
  draft,
  schema,
  slow,
  problems,
  onChange,
}: {
  draft: FormatDraft;
  schema: FormatsSchema;
  slow: Slow | undefined;
  problems: Problems;
  onChange: (next: FormatDraft) => void;
}) {
  const kinds = kindsIn(schema, draft.dialect);
  const [first] = kinds;
  return (
    <div className="flex flex-col gap-3">
      {slow !== undefined && slow.releases > 0 && <SlowWarning slow={slow} />}
      <p className="text-ink-1">
        {matchSentence(draft.conditions) ?? "Add a condition to say which Releases this matches."}
      </p>
      {draft.dialect !== "slipstream" && (
        <p className="text-meta text-ink-3 in-data-[density=touch]:text-body-touch">
          {`Written for ${appWords[draft.dialect]}, so its values are ${appWords[draft.dialect]}'s.`}
        </p>
      )}
      {draft.conditions.length > 0 && (
        <ConditionList draft={draft} kinds={kinds} problems={problems} onChange={onChange} />
      )}
      {problems["conditions"] !== undefined && <AlertLine text={problems["conditions"]} />}
      {first !== undefined && (
        <div>
          <Button
            icon={Add01Icon}
            onClick={() => {
              onChange({ ...draft, conditions: [...draft.conditions, blankCondition(first)] });
            }}
          >
            Add condition
          </Button>
        </div>
      )}
    </div>
  );
}
