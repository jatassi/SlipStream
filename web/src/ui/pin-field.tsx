import { Field } from "@base-ui/react/field";
import { OTPField } from "@base-ui/react/otp-field";
import { Alert02Icon } from "@hugeicons/core-free-icons";

import { errorText, useFieldContext } from "./form-context";
import { Icon } from "./icon";

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
    <Field.Root name={field.name} invalid={error !== undefined} className="flex flex-col gap-1.5">
      <Field.Label className="text-body font-semibold text-ink-2 in-data-[density=touch]:text-body-touch in-data-[density=touch]:font-semibold">
        {label}
      </Field.Label>
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
            className="h-(--input-height) w-10 rounded-item bg-well text-center text-body text-ink-1 tabular-nums ring-1 ring-hairline ring-inset in-data-[density=touch]:w-12 in-data-[density=touch]:text-input-touch"
          />
        ))}
      </OTPField.Root>
      {error === undefined ? null : (
        <Field.Error
          match
          className="flex items-center gap-1.5 text-meta text-fail-ink in-data-[density=touch]:text-body-touch"
        >
          <Icon icon={Alert02Icon} className="size-3.5 shrink-0" />
          {error}
        </Field.Error>
      )}
    </Field.Root>
  );
}
