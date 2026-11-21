import { Button } from "../ui/button";
import { useAppForm } from "../ui/form";
import type { RestoreFlow, RestoreStep } from "./flow";
import { SignalLine } from "../ui/signal-line";

export function PassphraseStep({
  step,
  flow,
}: {
  step: Extract<RestoreStep, { at: "passphrase" }>;
  flow: RestoreFlow;
}) {
  const form = useAppForm({
    defaultValues: { passphrase: "" },
    onSubmit: ({ value }) => flow.open(step.id, value.passphrase.trim()),
  });
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
        <form.AppField name="passphrase">
          {(field) => (
            <field.TextField
              label="Backup passphrase"
              description="The ten words from your recovery sheet."
              autoComplete="off"
              autoCapitalize="off"
              spellCheck={false}
            />
          )}
        </form.AppField>
        {step.refusal !== undefined && (
          <SignalLine tone="fail" announce>
            {step.refusal}
          </SignalLine>
        )}
        <div className="flex flex-wrap items-center gap-3">
          <form.SubmitButton label="Open the Backup" />
          <Button look="ghost" onClick={flow.reset}>
            Pick another file
          </Button>
        </div>
      </form.AppForm>
    </form>
  );
}
