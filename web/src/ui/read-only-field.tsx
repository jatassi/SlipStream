import { Field } from "@base-ui/react/field";
import { Input } from "@base-ui/react/input";

import { dataFont } from "./data";

export function ReadOnlyField({
  label,
  value,
  description,
}: {
  label: string;
  value: string;
  description: string;
}) {
  return (
    <Field.Root className="flex flex-col gap-1.5">
      <Field.Label className="text-body font-semibold text-ink-2 in-data-[density=touch]:text-body-touch in-data-[density=touch]:font-semibold">
        {label}
      </Field.Label>
      <Input
        readOnly
        value={value}
        onFocus={(event) => {
          event.currentTarget.select();
        }}
        className={`h-(--input-height) min-w-0 rounded-item bg-well px-3 ${dataFont} text-body text-ink-1 ring-1 ring-hairline ring-inset in-data-[density=touch]:text-input-touch`}
      />
      <Field.Description className="text-meta text-ink-3 in-data-[density=touch]:text-body-touch">
        {description}
      </Field.Description>
    </Field.Root>
  );
}
