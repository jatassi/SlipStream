import { Field } from "@base-ui/react/field";
import { OTPField } from "@base-ui/react/otp-field";

import { errorText, useFieldContext } from "./form-context";

const pinLength = 4;

export function PinField({
  label,
  autoComplete,
  autoSubmit = false,
}: {
  label: string;
  autoComplete: "current-password" | "new-password";
  autoSubmit?: boolean;
}) {
  const field = useFieldContext<string>();
  const error = errorText(field.state.meta.errors);
  return (
    <Field.Root name={field.name} invalid={error !== undefined} className="flex flex-col gap-1">
      <Field.Label className="text-sm font-semibold text-ink-3">{label}</Field.Label>
      <OTPField.Root
        length={pinLength}
        mask
        validationType="numeric"
        inputMode="numeric"
        autoComplete={autoComplete}
        autoSubmit={autoSubmit}
        value={field.state.value}
        onValueChange={(value) => {
          field.handleChange(value);
        }}
        className="flex gap-2"
      >
        {Array.from({ length: pinLength }, (_, slot) => (
          <OTPField.Input
            key={slot}
            aria-label={slot === 0 ? undefined : `${label} digit ${slot + 1} of ${pinLength}`}
            onBlur={field.handleBlur}
            className="size-11 rounded-md bg-fill text-center font-mono text-ink-1 ring-1 ring-hairline ring-inset"
          />
        ))}
      </OTPField.Root>
      {error === undefined ? null : (
        <Field.Error match className="text-sm text-fail-ink">
          {error}
        </Field.Error>
      )}
    </Field.Root>
  );
}
