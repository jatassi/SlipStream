import { Field } from "@base-ui/react/field";
import { OTPField } from "@base-ui/react/otp-field";
import { Alert02Icon } from "@hugeicons/core-free-icons";

import { errorText, useFieldContext } from "./form-context";
import { Icon } from "./icon";

export const pinLength = 4;

export type PinInputProps = {
  label: string;
  value: string;
  onValueChange: (value: string) => void;
  autoComplete: "current-password" | "new-password";
  error: string | undefined;
  name?: string;
  autoSubmit?: boolean;
  onComplete?: (value: string) => void;
  onBlur?: () => void;
  keypad?: boolean;
  className?: string;
};

export function PinInput({
  label,
  value,
  onValueChange,
  autoComplete,
  error,
  name,
  autoSubmit = false,
  onComplete,
  onBlur,
  keypad = false,
  className = "",
}: PinInputProps) {
  return (
    <Field.Root
      name={name}
      invalid={error !== undefined}
      className={`flex flex-col gap-1.5 ${className}`}
    >
      <Field.Label className="text-body font-semibold text-ink-2 in-data-[density=touch]:text-body-touch in-data-[density=touch]:font-semibold">
        {label}
      </Field.Label>
      <OTPField.Root
        length={pinLength}
        mask
        validationType="numeric"
        inputMode={keypad ? "none" : "numeric"}
        autoComplete={autoComplete}
        autoSubmit={autoSubmit}
        value={value}
        onValueChange={onValueChange}
        {...(onComplete && { onValueComplete: onComplete })}
        className="flex gap-2"
      >
        {Array.from({ length: pinLength }, (_, slot) => (
          <OTPField.Input
            key={slot}
            aria-label={slot === 0 ? undefined : `${label} digit ${slot + 1} of ${pinLength}`}
            onBlur={onBlur}
            className="h-(--input-height) w-10 rounded-item bg-well text-center text-body text-ink-1 tabular-nums ring-1 ring-hairline ring-inset in-data-[density=touch]:w-12 in-data-[density=touch]:text-input-touch"
          />
        ))}
      </OTPField.Root>
      <PinError error={error} />
    </Field.Root>
  );
}

function PinError({ error }: { error: string | undefined }) {
  return error === undefined ? null : (
    <Field.Error
      match
      className="flex items-center gap-1.5 text-meta text-fail-ink in-data-[density=touch]:text-body-touch"
    >
      <Icon icon={Alert02Icon} className="size-3.5 shrink-0" />
      {error}
    </Field.Error>
  );
}

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
  return (
    <PinInput
      label={label}
      name={field.name}
      value={field.state.value}
      onValueChange={(value) => {
        field.handleChange(value);
      }}
      onBlur={field.handleBlur}
      autoComplete={autoComplete}
      autoSubmit={autoSubmit}
      error={errorText(field.state.meta.errors)}
    />
  );
}
