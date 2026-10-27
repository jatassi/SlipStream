import { Field } from "@base-ui/react/field";
import type { ComponentProps } from "react";

import { dataFont } from "./data";
import { errorText, useFieldContext } from "./form-context";
import { FieldMessage } from "./text-field";

type AreaProps = Omit<ComponentProps<"textarea">, "value" | "onChange" | "onBlur" | "name">;

/** A text field that takes several lines, such as a pasted file or cookie. */
export function TextArea({
  label,
  description,
  data = false,
  rows = 6,
  ...area
}: AreaProps & {
  label: string;
  description?: string | undefined;
  /** Sets the text in Data, for a value someone pastes or compares, such as YAML. */
  data?: boolean;
}) {
  const field = useFieldContext<string>();
  const error = errorText(field.state.meta.errors);
  return (
    <Field.Root name={field.name} invalid={error !== undefined} className="flex flex-col gap-1.5">
      <Field.Label className="text-body font-semibold text-ink-2 in-data-[density=touch]:text-body-touch in-data-[density=touch]:font-semibold">
        {label}
      </Field.Label>
      <Field.Control
        render={<textarea {...area} rows={rows} />}
        value={field.state.value}
        onValueChange={field.handleChange}
        onBlur={field.handleBlur}
        className={`min-h-(--input-height) w-full resize-y rounded-item bg-well px-3 py-2 text-body text-ink-1 ring-1 ring-hairline ring-inset placeholder:text-ink-3 in-data-[density=touch]:text-input-touch ${data ? dataFont : ""}`}
      />
      <FieldMessage error={error} description={description} />
    </Field.Root>
  );
}
