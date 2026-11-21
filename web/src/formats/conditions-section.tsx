import { Add01Icon } from "@hugeicons/core-free-icons";

import type { FormatsSchema } from "../api/gen/model";
import { Button } from "../ui/button";
import { Callout } from "../ui/callout";
import { SignalLine } from "../ui/signal-line";
import { ConditionRow } from "./condition-row";
import { matchSentence, slowSentence } from "./condition-words";
import { type FormatDraft, type Kind, type Problems, blankCondition, kindsIn } from "./draft";
import { appWords } from "./paste-words";

export type Slow = { readonly releases: number; readonly limit: number };

function SlowWarning({ slow }: { slow: Slow }) {
  return (
    <Callout tone="warn" chip="Slow pattern">
      {slowSentence(slow.releases, slow.limit)}
    </Callout>
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
        {matchSentence(draft.conditions) ?? "Add a condition to say which releases this matches."}
      </p>
      {draft.dialect !== "slipstream" && (
        <p className="text-meta text-ink-3 in-data-[density=touch]:text-body-touch">
          {`This format was written for ${appWords[draft.dialect]}, and its conditions use ${appWords[draft.dialect]}'s values.`}
        </p>
      )}
      {draft.conditions.length > 0 && (
        <ConditionList draft={draft} kinds={kinds} problems={problems} onChange={onChange} />
      )}
      {problems["conditions"] !== undefined && (
        <SignalLine tone="fail" announce>
          {problems["conditions"]}
        </SignalLine>
      )}
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
