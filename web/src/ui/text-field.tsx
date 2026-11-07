import { Field } from "@base-ui/react/field";
import { Input } from "@base-ui/react/input";
import { Alert02Icon } from "@hugeicons/core-free-icons";
import type { ComponentProps, ReactNode } from "react";

import { dataFont } from "./data";
import { errorText, useFieldContext } from "./form-context";
import { Icon } from "./icon";

type InputProps = Omit<ComponentProps<"input">, "value" | "onChange" | "onBlur" | "name">;

export function TextField({
  label,
  format,
  description,
  data = false,
  action,
  ...input
}: InputProps & {
  label: string;
  format?: (value: string) => string;
  description?: string | undefined;
  /** Sets the value in Data, for a value someone copies or compares, such as a path. */
  data?: boolean;
  /** A control beside the input that acts on its value, such as Browse. */
  action?: ReactNode;
}) {
  const field = useFieldContext<string>();
  const error = errorText(field.state.meta.errors);
  return (
    <Field.Root name={field.name} invalid={error !== undefined} className="flex flex-col gap-1.5">
      <Field.Label className="text-body font-semibold text-ink-2 in-data-[density=touch]:text-body-touch in-data-[density=touch]:font-semibold">
        {label}
      </Field.Label>
      <div className="flex items-center gap-2">
        <Input
          {...input}
          value={field.state.value}
          onValueChange={(value) => {
            field.handleChange(format === undefined ? value : format(value));
          }}
          onBlur={field.handleBlur}
          className={`h-(--input-height) min-w-0 flex-1 rounded-item bg-well px-3 text-body text-ink-1 ring-1 ring-hairline ring-inset placeholder:text-ink-3 in-data-[density=touch]:text-input-touch ${data ? dataFont : ""}`}
        />
        {action}
      </div>
      <FieldMessage error={error} description={description} />
    </Field.Root>
  );
}

/** A field's error, or else its description, under its control. */
export function FieldMessage({
  error,
  description,
}: {
  error: string | undefined;
  description: string | undefined;
}) {
  if (error !== undefined) {
    return (
      <Field.Error
        match
        className="flex items-center gap-1.5 text-meta text-fail-ink in-data-[density=touch]:text-body-touch"
      >
        <Icon icon={Alert02Icon} className="size-3.5 shrink-0" />
        {error}
      </Field.Error>
    );
  }
  return description === undefined ? null : (
    <Field.Description className="text-meta text-ink-3 in-data-[density=touch]:text-body-touch">
      {description}
    </Field.Description>
  );
}
