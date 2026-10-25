import { Field } from "@base-ui/react/field";
import { Input } from "@base-ui/react/input";
import { useState } from "react";

import {
  type ControlProps,
  type Draft,
  type Kind,
  SaveButton,
  defaultHint,
  inputLook,
  labelLook,
  onSubmit,
  unitLook,
} from "./setting-kind";

type Parts = { readonly address: string; readonly port: string };

/** Splits text such as 0.0.0.0:6881 or [::]:6881 at its last colon. */
function partsOf(text: string): Parts {
  const at = text.lastIndexOf(":");
  const address = text.slice(0, at);
  return {
    address: address.startsWith("[") && address.endsWith("]") ? address.slice(1, -1) : address,
    port: text.slice(at + 1),
  };
}

const textOf = ({ address, port }: Parts): string =>
  address.includes(":") ? `[${address}]:${port}` : `${address}:${port}`;

const highestPort = 65_535;

function parseAddress(parts: Parts, example: string): Draft<"address"> {
  const address = parts.address.trim();
  const port = parts.port.trim();
  if (address === "") {
    return { refused: `An address and port, such as ${example}.` };
  }
  if (!/^\d+$/u.test(port) || Number(port) < 1 || Number(port) > highestPort) {
    return { refused: "A port between 1 and 65535." };
  }
  return { value: textOf({ address, port: String(Number(port)) }) };
}

function PartInput({
  label,
  value,
  invalid,
  disabled,
  inputMode,
  className,
  onChange,
}: {
  label: string;
  value: string;
  invalid: boolean;
  disabled: boolean;
  inputMode: "numeric" | "text";
  className: string;
  onChange: (value: string) => void;
}) {
  return (
    <Field.Root invalid={invalid} disabled={disabled} className="flex items-center gap-2">
      <Field.Label className={unitLook}>{label}</Field.Label>
      <Input
        inputMode={inputMode}
        autoComplete="off"
        spellCheck={false}
        value={value}
        onValueChange={onChange}
        className={`${className} ${inputLook}`}
      />
    </Field.Root>
  );
}

function AddressControl({
  field,
  busy,
  invalid,
  disabled,
  submit,
  onEdit,
}: ControlProps<"address">) {
  const [parts, setParts] = useState(() => partsOf(field.value));
  const edit = (next: Parts) => {
    setParts(next);
    onEdit();
  };
  return (
    <form
      onSubmit={onSubmit(() => {
        submit(parseAddress(parts, field.default));
      })}
    >
      <fieldset>
        <legend className={`mb-1.5 ${labelLook}`}>{field.label}</legend>
        <div className="flex flex-wrap items-center gap-x-3 gap-y-2">
          <PartInput
            label="Address"
            value={parts.address}
            invalid={invalid}
            disabled={disabled}
            inputMode="text"
            className="w-40"
            onChange={(address) => {
              edit({ ...parts, address });
            }}
          />
          <PartInput
            label="Port"
            value={parts.port}
            invalid={invalid}
            disabled={disabled}
            inputMode="numeric"
            className="w-20"
            onChange={(port) => {
              edit({ ...parts, port });
            }}
          />
          <SaveButton shown={textOf(parts) !== field.value} busy={busy} />
        </div>
      </fieldset>
    </form>
  );
}

export const addressKind: Kind<"address"> = {
  words: (field) => field.value,
  hints: (field) => [defaultHint(field, field.default)],
  Control: AddressControl,
};
