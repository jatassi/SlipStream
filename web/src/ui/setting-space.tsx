import { useState } from "react";

import type { SettingAmount, SettingSpaceField } from "../api/gen/model";
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
} from "./setting-kind";

const gb = 2 ** 30;
const tb = 2 ** 40;

const spaceUnits = [
  { label: "GB", unit: "bytes", size: gb },
  { label: "TB", unit: "bytes", size: tb },
  { label: "%", unit: "percent", size: 1 },
] as const;

type SpaceUnit = (typeof spaceUnits)[number];

export function spaceWords({ amount, unit }: SettingAmount): string {
  if (unit === "percent") {
    return `${amount}%`;
  }
  const [size, name] = amount >= tb ? [tb, "TB"] : [gb, "GB"];
  return `${Math.round((amount / size) * 10) / 10} ${name}`;
}

function unitFor({ amount, unit }: SettingAmount): SpaceUnit {
  if (unit === "percent") {
    return spaceUnits[2];
  }
  return amount > 0 && amount % tb === 0 ? spaceUnits[1] : spaceUnits[0];
}

const percentRange = "Between 0 and 100%.";

const maxWords = (field: SettingSpaceField) =>
  spaceWords({ amount: field.maxBytes, unit: "bytes" });

const bytesRange = (field: SettingSpaceField) => `Between 0 GB and ${maxWords(field)}.`;

function parseSpace(field: SettingSpaceField, text: string, unit: SpaceUnit): Draft<"space"> {
  const typed = Number(text);
  const range = unit.unit === "percent" ? percentRange : bytesRange(field);
  if (text.trim() === "" || !Number.isFinite(typed) || typed < 0) {
    return { refused: range };
  }
  if (unit.unit === "percent") {
    if (!Number.isInteger(typed)) {
      return { refused: "In whole percent." };
    }
    return typed > 100 ? { refused: range } : { value: { amount: typed, unit: "percent" } };
  }
  const amount = Math.round(typed * unit.size);
  return amount > field.maxBytes ? { refused: range } : { value: { amount, unit: "bytes" } };
}

function SpaceControl({ field, busy, invalid, disabled, submit, onEdit }: ControlProps<"space">) {
  const [unit, setUnit] = useState(() => unitFor(field.value));
  const [text, setText] = useState(() => String(field.value.amount / unit.size));
  const changed =
    text.trim() === "" ||
    unit.unit !== field.value.unit ||
    Number(text) * unit.size !== field.value.amount;
  return (
    <form
      onSubmit={onSubmit(() => {
        submit(parseSpace(field, text, unit));
      })}
      className={controlRow}
    >
      <AmountInput
        label={field.label}
        value={text}
        invalid={invalid}
        disabled={disabled}
        inputMode="decimal"
        onChange={(value) => {
          setText(value);
          onEdit();
        }}
      />
      <Segmented
        label={`${field.label} unit`}
        options={spaceUnits.map(({ label }) => ({ value: label, label }))}
        value={unit.label}
        disabled={disabled}
        onValueChange={(label) => {
          setUnit(spaceUnits.find((candidate) => candidate.label === label) ?? unit);
          onEdit();
        }}
      />
      <SaveButton shown={changed} busy={busy} />
    </form>
  );
}

export const spaceKind: Kind<"space"> = {
  words: (field) => spaceWords(field.value),
  hints: (field) => [
    `${defaultHint(field, spaceWords(field.default))} Up to ${maxWords(field)} or 100%.`,
  ],
  Control: SpaceControl,
};
