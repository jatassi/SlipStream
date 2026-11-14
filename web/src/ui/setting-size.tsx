import { useState } from "react";

import type { SettingSizeField } from "../api/gen/model";
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

const mib = 2 ** 20;
const gib = 2 ** 30;

const sizeUnits = [
  { bytes: gib, word: "GB" },
  { bytes: mib, word: "MB" },
] as const;

type SizeUnit = (typeof sizeUnits)[number];

/** Bytes in GB when whole, else MB. */
export const sizeWords = (bytes: number): string =>
  bytes > 0 && bytes % gib === 0 ? `${bytes / gib} GB` : `${bytes / mib} MB`;

const sizeRange = (field: SettingSizeField) =>
  `Between ${sizeWords(field.min)} and ${sizeWords(field.max)}.`;

const sizeUnitFor = (bytes: number): SizeUnit =>
  bytes > 0 && bytes % gib === 0 ? sizeUnits[0] : sizeUnits[1];

function parseSize(field: SettingSizeField, amount: string, unit: SizeUnit): Draft<"size"> {
  const bytes = Number(amount) * unit.bytes;
  if (amount.trim() === "" || !Number.isFinite(bytes) || bytes < field.min || bytes > field.max) {
    return { refused: sizeRange(field) };
  }
  return Number.isInteger(bytes) && bytes % mib === 0
    ? { value: bytes }
    : { refused: "In whole MB." };
}

function SizeControl({ field, busy, invalid, disabled, submit, onEdit }: ControlProps<"size">) {
  const [unit, setUnit] = useState(() => sizeUnitFor(field.value));
  const [amount, setAmount] = useState(() => String(field.value / unit.bytes));
  return (
    <form
      onSubmit={onSubmit(() => {
        submit(parseSize(field, amount, unit));
      })}
      className={controlRow}
    >
      <AmountInput
        label={field.label}
        value={amount}
        invalid={invalid}
        disabled={disabled}
        inputMode="decimal"
        onChange={(value) => {
          setAmount(value);
          onEdit();
        }}
      />
      <Segmented
        label={`${field.label} unit`}
        options={sizeUnits.map((candidate) => ({
          value: candidate.word,
          label: candidate.word,
        }))}
        value={unit.word}
        disabled={disabled}
        onValueChange={(word) => {
          setUnit(sizeUnits.find((candidate) => candidate.word === word) ?? unit);
          onEdit();
        }}
      />
      <SaveButton
        shown={amount.trim() === "" || Number(amount) * unit.bytes !== field.value}
        busy={busy}
      />
    </form>
  );
}

export const sizeKind: Kind<"size"> = {
  words: (field) => sizeWords(field.value),
  hints: (field) => [`${defaultHint(field, sizeWords(field.default))} ${sizeRange(field)}`],
  Control: SizeControl,
};
