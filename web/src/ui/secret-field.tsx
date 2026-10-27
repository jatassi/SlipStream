import { Field } from "@base-ui/react/field";
import { Input } from "@base-ui/react/input";
import { useEffect, useRef, useState } from "react";

import { Button } from "./button";
import { Data, dataFont } from "./data";
import { errorText, useFieldContext } from "./form-context";
import { FieldMessage } from "./text-field";

const labelLook =
  "text-body font-semibold text-ink-2 in-data-[density=touch]:text-body-touch in-data-[density=touch]:font-semibold";
const wellLook =
  "h-(--input-height) min-w-0 flex-1 rounded-item bg-well px-3 text-body ring-1 ring-hairline ring-inset in-data-[density=touch]:text-input-touch";

type Props = {
  label: string;
  saved: boolean;
  /** What of the saved secret may be shown, such as a feed's host. */
  shown?: string | undefined;
  type?: "password" | "url";
  description?: string | undefined;
};

function Saved({
  label,
  shown,
  error,
  description,
  onReplace,
}: Pick<Props, "label" | "shown" | "description"> & {
  error: string | undefined;
  onReplace: () => void;
}) {
  return (
    <fieldset className="flex flex-col gap-1.5">
      <legend className={`mb-1.5 ${labelLook}`}>{label}</legend>
      <div className="flex items-center gap-2">
        <span className={`flex items-center gap-2 text-ink-2 ${wellLook}`}>
          <span>Saved</span>
          {shown === undefined ? null : <Data className="truncate text-ink-3">{shown}</Data>}
        </span>
        <Button aria-label={`Replace ${label}`} onClick={onReplace}>
          Replace
        </Button>
      </div>
      <Field.Root className="contents">
        <FieldMessage error={error} description={description} />
      </Field.Root>
    </fieldset>
  );
}

/**
 * A secret the server never sends back. Once saved it reads "Saved", with
 * what may be shown of it such as a feed's host, until Replace opens an empty
 * input. The field's value stays "" unless a new secret is typed, so a save
 * sends only a replaced one.
 */
export function SecretField({ label, saved, shown, type = "password", description }: Props) {
  const field = useFieldContext<string>();
  const [replacing, setReplacing] = useState(!saved);
  const input = useRef<HTMLInputElement>(null);
  useEffect(() => {
    if (saved && replacing) {
      input.current?.focus();
    }
  }, [saved, replacing]);
  const error = errorText(field.state.meta.errors);
  if (!replacing) {
    const replace = () => {
      setReplacing(true);
    };
    return <Saved {...{ label, shown, error, description }} onReplace={replace} />;
  }
  return (
    <Field.Root name={field.name} invalid={error !== undefined} className="flex flex-col gap-1.5">
      <Field.Label className={labelLook}>{label}</Field.Label>
      <div className="flex items-center gap-2">
        <Input
          ref={input}
          type={type}
          autoComplete="new-password"
          spellCheck={false}
          value={field.state.value}
          onValueChange={field.handleChange}
          onBlur={field.handleBlur}
          className={`text-ink-1 ${wellLook} ${type === "url" ? dataFont : ""}`}
        />
        {saved && (
          <Button
            look="ghost"
            onClick={() => {
              field.handleChange("");
              setReplacing(false);
            }}
          >
            Cancel
          </Button>
        )}
      </div>
      <FieldMessage error={error} description={description} />
    </Field.Root>
  );
}
