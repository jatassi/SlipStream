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

/** An IP address as typed, blank to empty it; the setting's owner judges it on save. */
function IPControl({ field, busy, invalid, disabled, submit, onEdit }: ControlProps<"ip">) {
  const saved = field.value ?? "";
  const [text, setText] = useState(saved);
  return (
    <form
      onSubmit={onSubmit(() => {
        submit({ value: text.trim() });
      })}
      className="flex flex-wrap items-end gap-2"
    >
      <Field.Root invalid={invalid} disabled={disabled} className="flex flex-col gap-1.5">
        <Field.Label className={labelLook}>{field.label}</Field.Label>
        <Input
          inputMode="text"
          autoComplete="off"
          spellCheck={false}
          value={text}
          onValueChange={(next) => {
            setText(next);
            onEdit();
          }}
          className={`w-72 max-w-full ${inputLook}`}
        />
      </Field.Root>
      <SaveButton shown={text.trim() !== saved} busy={busy} />
    </form>
  );
}

export const ipKind: Kind<"ip"> = {
  words: (field) => field.value ?? "None",
  hints: () => ["Empty by default."],
  Control: IPControl,
};

function parseFile(path: string): Draft<"file"> {
  const file = path.trim();
  return file === "" ? { refused: "Choose a file." } : { value: file };
}

/** A program's path on the setting's Node, which judges it on save. */
function FileControl({ field, busy, invalid, disabled, submit, onEdit }: ControlProps<"file">) {
  const [text, setText] = useState(field.value);
  return (
    <form
      onSubmit={onSubmit(() => {
        submit(parseFile(text));
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
      <SaveButton shown={text.trim() !== field.value} busy={busy} />
    </form>
  );
}

export const fileKind: Kind<"file"> = {
  words: (field) => (field.value === "" ? "Bundled" : field.value),
  hints: (field) => [
    "Empty by default, which runs the one SlipStream bundles.",
    ...(field.unsupported ? ["Unsupported."] : []),
  ],
  Control: FileControl,
};
