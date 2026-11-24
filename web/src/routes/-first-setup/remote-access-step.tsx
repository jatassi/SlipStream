import { useState } from "react";

import type { ReachabilitySuggestionView } from "../../api/gen/model";
import {
  addExposure,
  runRemoteAccess,
  useGetRemoteAccessSuspense,
} from "../../api/gen/query/reachability/reachability";
import { Button } from "../../ui/button";
import { useAppForm } from "../../ui/form";
import { type SubmitErrors, submitErrors } from "../../ui/form-context";
import { Switch } from "../../ui/switch";
import { StepFooter, type StepNav } from "./frame";
import type { StepBodyProps } from "./screens";

const idOf = (suggestion: ReachabilitySuggestionView) => suggestion.serverId ?? "ui-portal";

const fieldOf = (suggestion: ReachabilitySuggestionView): `public.${string}` =>
  `public.${idOf(suggestion)}`;

const needed = "Enter the address players outside your home use";

type Publics = Readonly<Record<string, string>>;

// Every chosen suggestion is added; the first refusal names its field.
async function addEach(
  chosen: readonly ReachabilitySuggestionView[],
  publics: Publics,
): Promise<SubmitErrors | undefined> {
  const added = await Promise.allSettled(
    chosen.map((suggestion) =>
      addExposure(
        suggestion.kind === "media-server"
          ? {
              kind: "media-server",
              serverId: idOf(suggestion),
              public: publics[idOf(suggestion)] ?? "",
            }
          : { kind: "ui-portal" },
      ),
    ),
  );
  const index = added.findIndex((result) => result.status === "rejected");
  const refused = added[index];
  const suggestion = chosen[index];
  if (refused?.status !== "rejected" || suggestion === undefined) {
    return undefined;
  }
  return submitErrors(refused.reason, {
    public_required: fieldOf(suggestion),
    public_not_url: fieldOf(suggestion),
  });
}

function missingOf(chosen: readonly ReachabilitySuggestionView[], publics: Publics) {
  const missing = chosen.filter(
    (suggestion) =>
      suggestion.kind === "media-server" && (publics[idOf(suggestion)] ?? "").trim() === "",
  );
  return missing.length === 0
    ? undefined
    : { fields: Object.fromEntries(missing.map((suggestion) => [fieldOf(suggestion), needed])) };
}

// Continue adds each switched-on suggestion, checks them, then marks the step done.
function useStepForm(suggestions: readonly ReachabilitySuggestionView[], nav: StepNav) {
  const [off, setOff] = useState<ReadonlySet<string>>(new Set());
  const chosen = suggestions.filter((suggestion) => !off.has(idOf(suggestion)));
  const form = useAppForm({
    defaultValues: {
      public: Object.fromEntries(suggestions.map((each) => [idOf(each), each.public])),
    },
    onSubmit: async ({ value, formApi }) => {
      const refused = missingOf(chosen, value.public) ?? (await addEach(chosen, value.public));
      if (refused !== undefined) {
        formApi.setErrorMap({ onSubmit: refused });
        return;
      }
      if (chosen.length > 0) {
        await runRemoteAccess();
      }
      await nav.complete();
    },
  });
  const turn = (id: string, on: boolean) => {
    const next = new Set(off);
    if (on) {
      next.delete(id);
    } else {
      next.add(id);
    }
    setOff(next);
  };
  return { form, off, turn };
}

type StepForm = ReturnType<typeof useStepForm>;

function SuggestionRow({
  suggestion,
  step,
}: {
  suggestion: ReachabilitySuggestionView;
  step: StepForm;
}) {
  const id = idOf(suggestion);
  const on = !step.off.has(id);
  return (
    <li className="flex flex-col gap-2">
      <Switch
        label={suggestion.subject}
        checked={on}
        onCheckedChange={(checked) => {
          step.turn(id, checked);
        }}
      />
      {on && suggestion.kind === "media-server" && (
        <step.form.AppField name={fieldOf(suggestion)}>
          {(field) => (
            <field.TextField
              label="Public address"
              type="url"
              autoComplete="off"
              spellCheck={false}
              data
              description={`The address players outside your home use for ${suggestion.subject}.`}
            />
          )}
        </step.form.AppField>
      )}
    </li>
  );
}

/** Each suggestion with a switch. A UI and portal outsiders can't use yet waits for the Address step. */
export function RemoteAccessStep({ nav }: StepBodyProps) {
  const suggestions = useGetRemoteAccessSuspense().data.suggestions.filter(
    (suggestion) => !suggestion.addressPrivate,
  );
  const step = useStepForm(suggestions, nav);
  const { form } = step;
  return (
    <form
      noValidate
      className="flex flex-col gap-4"
      onSubmit={(event) => {
        event.preventDefault();
        void form.handleSubmit();
      }}
    >
      <form.AppForm>
        {suggestions.length === 0 ? (
          <p>
            Everything SlipStream suggested is set up. Nodes › Remote access shows whether each one
            works from outside your home.
          </p>
        ) : (
          <ul className="flex flex-col gap-4">
            {suggestions.map((suggestion) => (
              <SuggestionRow key={idOf(suggestion)} suggestion={suggestion} step={step} />
            ))}
          </ul>
        )}
        <form.FormAlert />
        <StepFooter
          nav={nav}
          primary={
            <form.Subscribe selector={(state) => state.isSubmitting}>
              {(submitting) => (
                <Button type="submit" look="primary" busy={submitting}>
                  Continue
                </Button>
              )}
            </form.Subscribe>
          }
        />
      </form.AppForm>
    </form>
  );
}
