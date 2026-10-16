import { Field } from "@base-ui/react/field";
import { Input } from "@base-ui/react/input";

import { Button } from "./button";
import { dataFont } from "./data";
import { toast } from "./toast";

async function copyOut(label: string, value: string) {
  try {
    await navigator.clipboard.writeText(value);
  } catch {
    toast(`Couldn't copy the ${label.toLowerCase()}`, { kind: "fail" });
    return;
  }
  toast(`${label} copied`);
}

// The Clipboard API exists only in a secure context, and a Node on a LAN
// often answers plain http, where the field's select-on-focus still copies.
export function ReadOnlyField({
  label,
  value,
  description,
  copy = false,
}: {
  label: string;
  value: string;
  description?: string;
  copy?: boolean;
}) {
  return (
    <Field.Root className="flex flex-col gap-1.5">
      <Field.Label className="text-body font-semibold text-ink-2 in-data-[density=touch]:text-body-touch in-data-[density=touch]:font-semibold">
        {label}
      </Field.Label>
      <div className="flex items-center gap-2">
        <Input
          readOnly
          value={value}
          onFocus={(event) => {
            event.currentTarget.select();
          }}
          className={`h-(--input-height) min-w-0 flex-1 rounded-item bg-well px-3 ${dataFont} text-body text-ink-1 ring-1 ring-hairline ring-inset in-data-[density=touch]:text-input-touch`}
        />
        {copy && window.isSecureContext && (
          <Button
            onClick={() => {
              void copyOut(label, value);
            }}
          >
            Copy
          </Button>
        )}
      </div>
      {description !== undefined && (
        <Field.Description className="text-meta text-ink-3 in-data-[density=touch]:text-body-touch">
          {description}
        </Field.Description>
      )}
    </Field.Root>
  );
}
