import { Field } from "@base-ui/react/field";
import { Input } from "@base-ui/react/input";
import type { ReactNode } from "react";

import type {
  SettingChoiceField,
  SettingCountField,
  SettingDurationField,
  SettingTimes,
  SettingWindowField,
} from "../api/gen/model";
import { Button } from "./button";

export type Fields = {
  duration: SettingDurationField;
  count: SettingCountField;
  choice: SettingChoiceField;
  window: SettingWindowField;
};

export type Values = {
  duration: number;
  count: number;
  choice: string;
  window: SettingTimes;
};

export type SettingKind = keyof Fields;

export type Setting = Fields[SettingKind];

/** A value the field accepted, or the sentence that refuses it. */
export type Draft<K extends SettingKind> = { value: Values[K] } | { refused: string };

export type ControlProps<K extends SettingKind> = {
  field: Fields[K];
  busy: boolean;
  invalid: boolean;
  submit: (draft: Draft<K>) => void;
  onEdit: () => void;
};

/** What a kind of setting brings: its value in words, its hints and its control. */
export type Kind<K extends SettingKind> = {
  words: (field: Fields[K]) => string;
  hints: (field: Fields[K]) => readonly string[];
  Control: (props: ControlProps<K>) => ReactNode;
};

export const labelLook =
  "text-body font-semibold text-ink-2 in-data-[density=touch]:text-body-touch in-data-[density=touch]:font-semibold";

export const inputLook =
  "h-(--input-height) rounded-item bg-well px-3 text-body text-ink-1 tabular-nums ring-1 ring-hairline ring-inset aria-invalid:ring-fail-line in-data-[density=touch]:text-input-touch";

export const controlRow =
  "grid grid-cols-[auto_auto_minmax(0,1fr)] items-center justify-items-start gap-x-2 gap-y-1.5";

export const unitLook = "text-body text-ink-2 in-data-[density=touch]:text-body-touch";

export const defaultHint = (field: Setting, words: string): string =>
  field.defaultFrom === undefined
    ? `Default ${words}.`
    : `Default ${words}, from ${field.defaultFrom}.`;

export function SaveButton({ shown, busy }: { shown: boolean; busy: boolean }) {
  return shown || busy ? (
    <Button type="submit" size="small" busy={busy}>
      Save
    </Button>
  ) : null;
}

export function onSubmit(submit: () => void) {
  return (event: { preventDefault: () => void }) => {
    event.preventDefault();
    submit();
  };
}

// The label spans the row above the input, and the controls after the input
// stay outside the Field so its label names the input alone.
export function AmountInput({
  label,
  value,
  invalid,
  inputMode,
  onChange,
}: {
  label: string;
  value: string;
  invalid: boolean;
  inputMode: "decimal" | "numeric";
  onChange: (value: string) => void;
}) {
  return (
    <Field.Root invalid={invalid} className="contents">
      <Field.Label className={`col-span-full ${labelLook}`}>{label}</Field.Label>
      <Input
        inputMode={inputMode}
        value={value}
        onValueChange={onChange}
        className={`w-20 ${inputLook}`}
      />
    </Field.Root>
  );
}
