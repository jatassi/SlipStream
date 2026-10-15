import { Field } from "@base-ui/react/field";
import { Input } from "@base-ui/react/input";
import type { ComponentProps } from "react";

import { errorText, useFieldContext } from "./form-context";

type InputProps = Omit<ComponentProps<"input">, "value" | "onChange" | "onBlur" | "name">;

export function TextField({
  label,
  format,
  description,
  ...input
}: InputProps & {
  label: string;
  format?: (value: string) => string;
  description?: string | undefined;
}) {
  const field = useFieldContext<string>();
  const error = errorText(field.state.meta.errors);
  return (
    <Field.Root name={field.name} invalid={error !== undefined} className="flex flex-col gap-1">
      <Field.Label className="text-sm font-semibold text-ink-3">{label}</Field.Label>
      <Input
        {...input}
        value={field.state.value}
        onValueChange={(value) => {
          field.handleChange(format === undefined ? value : format(value));
        }}
        onBlur={field.handleBlur}
        className="rounded-md bg-fill px-3 py-2 text-ink-1 ring-1 ring-hairline ring-inset"
      />
      {error === undefined ? (
        description === undefined ? null : (
          <Field.Description>{description}</Field.Description>
        )
      ) : (
        <Field.Error match className="text-sm text-fail-ink">
          {error}
        </Field.Error>
      )}
    </Field.Root>
  );
}
