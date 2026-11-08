import type { ReactNode } from "react";

import type { TrackersRule, TrackersRuleBody } from "../api/gen/model";
import { useAppForm } from "../ui/form";
import { type Clears, ruleBodyOf, ruleCounts, ruleDraftOf, seedingRefusal } from "./draft";
import { Choice, EditorActions, EditorForm } from "./editor-parts";
import { PartsFields } from "./parts-fields";

const clearsOptions: readonly { value: Clears; label: string }[] = [
  { value: "time", label: "Seeding time" },
  { value: "ratio", label: "Ratio" },
  { value: "either", label: "Either one" },
  { value: "both", label: "Both" },
];

function useRuleForm(
  rule: TrackersRule | null,
  onSave: (rule: TrackersRuleBody) => Promise<unknown>,
) {
  return useAppForm({
    defaultValues: ruleDraftOf(rule),
    onSubmit: async ({ value, formApi }) => {
      try {
        await onSave(ruleBodyOf(value));
      } catch (error) {
        formApi.setErrorMap({ onSubmit: seedingRefusal(error) });
      }
    },
  });
}

/** A Tracker's seeding rule, saved whole. `aside` sits beside Save. */
export function RuleEditor({
  rule,
  onSave,
  aside,
}: {
  rule: TrackersRule | null;
  onSave: (rule: TrackersRuleBody) => Promise<unknown>;
  aside?: ReactNode;
}) {
  const form = useRuleForm(rule, onSave);
  return (
    <EditorForm label="Seeding rule" submit={() => form.handleSubmit()}>
      <form.AppForm>
        <form.Field name="clears">
          {(field) => (
            <Choice
              label="Met by"
              options={clearsOptions}
              value={field.state.value}
              onValueChange={field.handleChange}
            />
          )}
        </form.Field>
        <form.Subscribe selector={(state) => ruleCounts[state.values.clears]}>
          {(counts) => (
            <PartsFields
              form={form}
              fields="parts"
              ratio={counts.ratio}
              time={counts.time}
              optional={false}
            />
          )}
        </form.Subscribe>
        <EditorActions save="Save rule" aside={aside} />
      </form.AppForm>
    </EditorForm>
  );
}
