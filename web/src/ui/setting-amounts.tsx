import { useState } from "react";

import type { SettingCountField, SettingDurationField } from "../api/gen/model";
import { countWords, durationWords } from "../format";
import { Segmented } from "./segmented";
import {
  AmountInput,
  type ControlProps,
  type Draft,
  type Kind,
  SaveButton,
  controlRow,
  defaultHint,
  onSubmit,
  unitLook,
} from "./setting-kind";

const durationUnits = [
  { seconds: 86_400, label: "days" },
  { seconds: 3600, label: "h" },
  { seconds: 60, label: "min" },
  { seconds: 1, label: "s" },
] as const;

type DurationUnit = (typeof durationUnits)[number];

const stepWords: Readonly<Record<number, string>> = {
  1: "seconds",
  60: "minutes",
  3600: "hours",
  86_400: "days",
};

// A unit is offered when a whole number of it is a whole number of steps,
// and it fits under the maximum.
function unitsOf(field: SettingDurationField): readonly DurationUnit[] {
  return durationUnits.filter(
    (unit) =>
      unit.seconds % field.step === 0 && (unit.seconds <= field.max || unit.seconds === field.step),
  );
}

const unitFor = (seconds: number, units: readonly DurationUnit[]): DurationUnit =>
  units.find((unit) => seconds > 0 && seconds % unit.seconds === 0) ??
  units.at(-1) ??
  durationUnits[3];

const durationRange = (field: SettingDurationField) =>
  `Between ${durationWords(field.min)} and ${durationWords(field.max)}.`;

function parseDuration(
  field: SettingDurationField,
  amount: string,
  unit: DurationUnit,
): Draft<"duration"> {
  const seconds = Math.round(Number(amount) * unit.seconds * 1000) / 1000;
  if (amount.trim() === "" || !Number.isFinite(seconds)) {
    return { refused: durationRange(field) };
  }
  if (seconds < field.min && field.floor !== undefined) {
    return { refused: `No faster than ${durationWords(field.min)}. ${field.floor}` };
  }
  if (seconds < field.min || seconds > field.max) {
    return { refused: durationRange(field) };
  }
  if (!Number.isInteger(seconds) || seconds % field.step !== 0) {
    return { refused: `In whole ${stepWords[field.step] ?? "steps"}.` };
  }
  return { value: seconds };
}

function UnitPicker({
  field,
  units,
  unit,
  onPick,
}: {
  field: SettingDurationField;
  units: readonly DurationUnit[];
  unit: DurationUnit;
  onPick: (unit: DurationUnit) => void;
}) {
  return units.length === 1 ? (
    <span className={unitLook}>{unit.label}</span>
  ) : (
    <Segmented
      label={`${field.label} unit`}
      options={units.map(({ label }) => ({ value: label, label }))}
      value={unit.label}
      onValueChange={(label) => {
        onPick(units.find((candidate) => candidate.label === label) ?? unit);
      }}
    />
  );
}

function DurationControl({ field, busy, invalid, submit, onEdit }: ControlProps<"duration">) {
  const units = unitsOf(field);
  const [unit, setUnit] = useState(() => unitFor(field.value, units));
  const [amount, setAmount] = useState(() => String(field.value / unit.seconds));
  return (
    <form
      onSubmit={onSubmit(() => {
        submit(parseDuration(field, amount, unit));
      })}
      className={controlRow}
    >
      <AmountInput
        label={field.label}
        value={amount}
        invalid={invalid}
        inputMode="decimal"
        onChange={(value) => {
          setAmount(value);
          onEdit();
        }}
      />
      <UnitPicker
        field={field}
        units={units}
        unit={unit}
        onPick={(picked) => {
          setUnit(picked);
          onEdit();
        }}
      />
      <SaveButton
        shown={amount.trim() === "" || Number(amount) * unit.seconds !== field.value}
        busy={busy}
      />
    </form>
  );
}

export const durationKind: Kind<"duration"> = {
  words: (field) => durationWords(field.value),
  hints: (field) => [
    `${defaultHint(field, durationWords(field.default))} ${durationRange(field)}`,
    ...(field.floor === undefined
      ? []
      : [`No faster than ${durationWords(field.min)}. ${field.floor}`]),
  ],
  Control: DurationControl,
};

const countRange = (field: SettingCountField) =>
  `Between ${field.min} and ${countWords(field.max, field.unit)}.`;

function parseCount(field: SettingCountField, amount: string): Draft<"count"> {
  if (!/^\d+$/u.test(amount.trim())) {
    return { refused: countRange(field) };
  }
  const count = Number(amount);
  if (count < field.min && field.floor !== undefined) {
    return { refused: `At least ${countWords(field.min, field.unit)}. ${field.floor}` };
  }
  return count < field.min || count > field.max ? { refused: countRange(field) } : { value: count };
}

function CountControl({ field, busy, invalid, submit, onEdit }: ControlProps<"count">) {
  const [amount, setAmount] = useState(String(field.value));
  return (
    <form
      onSubmit={onSubmit(() => {
        submit(parseCount(field, amount));
      })}
      className={controlRow}
    >
      <AmountInput
        label={field.label}
        value={amount}
        invalid={invalid}
        inputMode="numeric"
        onChange={(value) => {
          setAmount(value);
          onEdit();
        }}
      />
      <span className={unitLook}>{field.unit}</span>
      <SaveButton shown={amount.trim() !== String(field.value)} busy={busy} />
    </form>
  );
}

export const countKind: Kind<"count"> = {
  words: (field) => countWords(field.value, field.unit),
  hints: (field) => [
    `${defaultHint(field, countWords(field.default, field.unit))} ${countRange(field)}`,
    ...(field.floor === undefined
      ? []
      : [`At least ${countWords(field.min, field.unit)}. ${field.floor}`]),
  ],
  Control: CountControl,
};
