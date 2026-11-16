import { Field } from "@base-ui/react/field";
import { Input } from "@base-ui/react/input";
import { type RefObject, useEffect, useRef, useState } from "react";

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
  required?: boolean;
  description?: string | undefined;
  /** An optional secret: Remove sets the field to null, which a save sends to remove it. */
  removable?: boolean;
  /** A secret of several lines, such as one URL per line, typed in a text area of this many rows. */
  rows?: number | undefined;
};

function Removed({ label }: { label: string }) {
  const field = useFieldContext<string | null>();
  return (
    <div className="flex items-center gap-2">
      <span className={`flex items-center text-ink-2 ${wellLook}`}>Removed. Save to apply.</span>
      <Button
        aria-label={`Undo removing ${label}`}
        onClick={() => {
          field.handleChange("");
        }}
      >
        Undo
      </Button>
    </div>
  );
}

function Saved({
  label,
  shown,
  error,
  description,
  removable,
  onReplace,
}: Pick<Props, "label" | "shown" | "description"> & {
  error: string | undefined;
  removable: boolean;
  onReplace: () => void;
}) {
  const field = useFieldContext<string | null>();
  return (
    <fieldset className="flex flex-col gap-1.5">
      <legend className={`mb-1.5 ${labelLook}`}>{label}</legend>
      {field.state.value === null ? (
        <Removed label={label} />
      ) : (
        <div className="flex items-center gap-2">
          <span className={`flex items-center gap-2 text-ink-2 ${wellLook}`}>
            <span>Saved</span>
            {shown === undefined ? null : <Data className="truncate text-ink-3">{shown}</Data>}
          </span>
          <Button aria-label={`Replace ${label}`} onClick={onReplace}>
            Replace
          </Button>
          {removable && (
            <Button
              aria-label={`Remove ${label}`}
              onClick={() => {
                field.handleChange(null);
              }}
            >
              Remove
            </Button>
          )}
        </div>
      )}
      <Field.Root className="contents">
        <FieldMessage error={error} description={description} />
      </Field.Root>
    </fieldset>
  );
}

function SecretInput({
  input,
  type,
  required,
  rows,
}: {
  required: boolean | undefined;
  rows: number | undefined;
  input: RefObject<HTMLInputElement | null>;
  type: NonNullable<Props["type"]>;
}) {
  const field = useFieldContext<string | null>();
  if (rows === undefined) {
    return (
      <Input
        ref={input}
        type={type}
        aria-required={required}
        autoComplete="new-password"
        spellCheck={false}
        value={field.state.value ?? ""}
        onValueChange={field.handleChange}
        onBlur={field.handleBlur}
        className={`text-ink-1 ${wellLook} ${type === "url" ? dataFont : ""}`}
      />
    );
  }
  return (
    <Field.Control
      render={<textarea rows={rows} spellCheck={false} />}
      aria-required={required}
      value={field.state.value ?? ""}
      onValueChange={field.handleChange}
      onBlur={field.handleBlur}
      className={`min-w-0 flex-1 resize-y rounded-item bg-well px-3 py-2 text-body text-ink-1 ring-1 ring-hairline ring-inset in-data-[density=touch]:text-input-touch ${dataFont}`}
    />
  );
}

/**
 * A secret the server never sends back. Once saved it reads "Saved", with
 * what may be shown of it such as a feed's host, until Replace opens an empty
 * input. The field's value stays "" unless a new secret is typed, so a save
 * sends only a replaced one. A removable one also offers Remove, which sets
 * the value to null until Undo or a save.
 */
export function SecretField(props: Props) {
  const { label, saved, shown, type = "password", description, removable = false } = props;
  const field = useFieldContext<string | null>();
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
    return <Saved {...{ label, shown, error, description, removable }} onReplace={replace} />;
  }
  return (
    <Field.Root name={field.name} invalid={error !== undefined} className="flex flex-col gap-1.5">
      <Field.Label className={labelLook}>{label}</Field.Label>
      <div className="flex items-center gap-2">
        <SecretInput input={input} type={type} required={props.required} rows={props.rows} />
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
