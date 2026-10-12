import { Field } from "@base-ui/react/field";
import { Input } from "@base-ui/react/input";

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
    <Field.Root className="flex flex-col gap-1">
      <Field.Label className="text-sm font-semibold text-ink-3">{label}</Field.Label>
      <Input
        readOnly
        value={value}
        onFocus={(event) => {
          event.currentTarget.select();
        }}
        className="rounded-md bg-fill px-3 py-2 font-mono text-ink-1 ring-1 ring-hairline ring-inset"
      />
      <Field.Description>{description}</Field.Description>
    </Field.Root>
  );
}
