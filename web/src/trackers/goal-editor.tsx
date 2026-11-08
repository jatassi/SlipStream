import type { ReactNode } from "react";

import type { TrackersGoal } from "../api/gen/model";
import { useAppForm } from "../ui/form";
import { Select } from "../ui/select";
import { type Action, type Combine, goalBodyOf, goalDraftOf, seedingRefusal } from "./draft";
import { Choice, EditorActions, EditorForm } from "./editor-parts";
import { PartsFields } from "./parts-fields";
import { actionLabels, goalSentence } from "./words";

const combines: readonly { value: Combine; label: string }[] = [
  { value: "first", label: "Whichever first" },
  { value: "both", label: "Both" },
  { value: "forever", label: "Forever" },
];

const actions: readonly { value: Action; label: string }[] = [
  { value: "pause", label: actionLabels.pause },
  { value: "remove", label: actionLabels.remove },
  { value: "remove-delete", label: actionLabels["remove-delete"] },
];

function useGoalForm(goal: TrackersGoal, onSave: (goal: TrackersGoal) => Promise<unknown>) {
  return useAppForm({
    defaultValues: goalDraftOf(goal),
    onSubmit: async ({ value, formApi }) => {
      try {
        await onSave(goalBodyOf(value));
      } catch (error) {
        formApi.setErrorMap({ onSubmit: seedingRefusal(error) });
      }
    },
  });
}

/** How far a goal that ends seeds, and what happens once it is met. */
function GoalEnd({ form }: { form: ReturnType<typeof useGoalForm> }) {
  return (
    <form.Subscribe selector={(state) => state.values.combine !== "forever"}>
      {(ends) =>
        ends && (
          <>
            <PartsFields form={form} fields="parts" ratio time optional />
            <form.Field name="action">
              {(field) => (
                <div className="max-w-sm">
                  <Select
                    label="When the goal is met"
                    options={actions}
                    value={field.state.value}
                    onValueChange={field.handleChange}
                  />
                </div>
              )}
            </form.Field>
          </>
        )
      }
    </form.Subscribe>
  );
}

/**
 * A seeding goal, saved whole. The same editor sets the default goal and a
 * Tracker's own goal. `aside` sits beside Save, such as "Use the default goal".
 */
export function GoalEditor({
  goal,
  onSave,
  aside,
}: {
  goal: TrackersGoal;
  onSave: (goal: TrackersGoal) => Promise<unknown>;
  aside?: ReactNode;
}) {
  const form = useGoalForm(goal, onSave);
  return (
    <EditorForm label="Seeding goal" submit={() => form.handleSubmit()}>
      <form.AppForm>
        <p className="text-ink-1">{goalSentence(goal)}</p>
        <form.Field name="combine">
          {(field) => (
            <Choice
              label="Seed until"
              options={combines}
              value={field.state.value}
              onValueChange={field.handleChange}
            />
          )}
        </form.Field>
        <GoalEnd form={form} />
        <EditorActions save="Save goal" aside={aside} />
      </form.AppForm>
    </EditorForm>
  );
}
