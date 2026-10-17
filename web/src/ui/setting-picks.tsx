import { Field } from "@base-ui/react/field";
import { Input } from "@base-ui/react/input";
import { useState } from "react";

import type { SettingChoiceField, SettingTimes } from "../api/gen/model";
import { Segmented } from "./segmented";
import { Select } from "./select";
import { Switch } from "./switch";
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

const optionWords = (field: SettingChoiceField, value: string) =>
  field.options.find((option) => option.value === value)?.label ?? value;

const segmentsAtMost = 3;

function ChoiceControl({ field, busy, disabled, submit }: ControlProps<"choice">) {
  const choose = (value: string) => {
    if (value !== field.value) {
      submit({ value });
    }
  };
  return field.options.length > segmentsAtMost ? (
    <Select
      label={field.label}
      options={field.options}
      value={field.value}
      disabled={disabled || busy}
      onValueChange={choose}
    />
  ) : (
    <div className="flex flex-col items-start gap-1.5">
      <span aria-hidden="true" className={labelLook}>
        {field.label}
      </span>
      <Segmented
        label={field.label}
        options={field.options}
        value={field.value}
        disabled={disabled || busy}
        onValueChange={choose}
      />
    </div>
  );
}

export const choiceKind: Kind<"choice"> = {
  words: (field) => optionWords(field, field.value),
  hints: (field) => [defaultHint(field, optionWords(field, field.default))],
  Control: ChoiceControl,
};

const clockPattern = /^(?:[01]\d|2[0-3]):[0-5]\d$/u;

function parseWindow(times: SettingTimes): Draft<"window"> {
  if (!clockPattern.test(times.start) || !clockPattern.test(times.end)) {
    return { refused: "Times are 00:00 to 23:59." };
  }
  return times.start === times.end
    ? { refused: "The window needs an end after its start." }
    : { value: times };
}

const windowWords = ({ start, end }: SettingTimes) => `${start} to ${end}`;

function TimeInput({
  label,
  value,
  invalid,
  disabled,
  onChange,
}: {
  label: string;
  value: string;
  invalid: boolean;
  disabled: boolean;
  onChange: (value: string) => void;
}) {
  return (
    <Field.Root invalid={invalid} disabled={disabled} className="flex items-center gap-2">
      <Field.Label className={unitLook}>{label}</Field.Label>
      <Input
        type="time"
        value={value}
        onValueChange={onChange}
        className={`[color-scheme:dark] ${inputLook}`}
      />
    </Field.Root>
  );
}

function WindowControl({ field, busy, invalid, disabled, submit, onEdit }: ControlProps<"window">) {
  const [times, setTimes] = useState(field.value);
  const edit = (next: SettingTimes) => {
    setTimes(next);
    onEdit();
  };
  return (
    <form
      onSubmit={onSubmit(() => {
        submit(parseWindow(times));
      })}
    >
      <fieldset>
        <legend className={`mb-1.5 ${labelLook}`}>{field.label}</legend>
        <div className="flex flex-wrap items-center gap-x-3 gap-y-2">
          <TimeInput
            label="From"
            value={times.start}
            invalid={invalid}
            disabled={disabled}
            onChange={(start) => {
              edit({ ...times, start });
            }}
          />
          <TimeInput
            label="to"
            value={times.end}
            invalid={invalid}
            disabled={disabled}
            onChange={(end) => {
              edit({ ...times, end });
            }}
          />
          <span className="text-body text-ink-3 in-data-[density=touch]:text-body-touch">
            {field.zone}
          </span>
          <SaveButton
            shown={times.start !== field.value.start || times.end !== field.value.end}
            busy={busy}
          />
        </div>
      </fieldset>
    </form>
  );
}

export const windowKind: Kind<"window"> = {
  words: (field) => `${windowWords(field.value)} ${field.zone}`,
  hints: (field) => [defaultHint(field, windowWords(field.default))],
  Control: WindowControl,
};

const onOff = (on: boolean) => (on ? "On" : "Off");

function SwitchSetting({ field, busy, disabled, submit }: ControlProps<"switch">) {
  return (
    <Switch
      label={field.label}
      checked={field.value}
      disabled={disabled || busy}
      onCheckedChange={(on) => {
        submit({ value: on });
      }}
    />
  );
}

export const switchKind: Kind<"switch"> = {
  words: (field) => onOff(field.value),
  hints: (field) => [defaultHint(field, onOff(field.default))],
  Control: SwitchSetting,
};
