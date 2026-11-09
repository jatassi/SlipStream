import { Field } from "@base-ui/react/field";
import { Input } from "@base-ui/react/input";
import { useState } from "react";

import {
  type ControlProps,
  type Kind,
  SaveButton,
  defaultHint,
  inputLook,
  labelLook,
  onSubmit,
} from "./setting-kind";

/** Template text as typed; the setting's owner judges it on save and its refusal shows below. */
function TemplateControl({
  field,
  busy,
  invalid,
  disabled,
  submit,
  onEdit,
}: ControlProps<"template">) {
  const [text, setText] = useState(field.value);
  return (
    <form
      onSubmit={onSubmit(() => {
        submit({ value: text });
      })}
      className="flex flex-col items-start gap-1.5"
    >
      <Field.Root
        invalid={invalid}
        disabled={disabled}
        className="flex flex-col gap-1.5 self-stretch"
      >
        <Field.Label className={labelLook}>{field.label}</Field.Label>
        <Input
          autoComplete="off"
          spellCheck={false}
          value={text}
          onValueChange={(next) => {
            setText(next);
            onEdit();
          }}
          className={`w-full ${inputLook}`}
        />
      </Field.Root>
      <SaveButton shown={text !== field.value} busy={busy} />
    </form>
  );
}

export const templateKind: Kind<"template"> = {
  words: (field) => field.value,
  hints: (field) => [defaultHint(field, `"${field.default}"`)],
  Control: TemplateControl,
};
