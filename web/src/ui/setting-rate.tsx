import { useState } from "react";

import type { SettingRateField } from "../api/gen/model";
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

const kib = 1024;
const mib = 1024 * kib;

const rateUnits = [
  { bytes: mib, word: "MB/s" },
  { bytes: kib, word: "KB/s" },
] as const;

type RateUnit = (typeof rateUnits)[number];

/** Bytes a second in MB/s when whole, else KB/s. */
const rateWords = (perSecond: number): string =>
  perSecond > 0 && perSecond % mib === 0 ? `${perSecond / mib} MB/s` : `${perSecond / kib} KB/s`;

const rateAmount = (field: SettingRateField, perSecond: number): string =>
  perSecond === 0 && field.zero !== undefined ? field.zero : rateWords(perSecond);

const rateRange = (field: SettingRateField) => `Up to ${rateWords(field.max)}.`;

const rateUnitFor = (perSecond: number): RateUnit =>
  perSecond % mib === 0 ? rateUnits[0] : rateUnits[1];

function parseRate(field: SettingRateField, amount: string, unit: RateUnit): Draft<"rate"> {
  const perSecond = Number(amount) * unit.bytes;
  if (
    amount.trim() === "" ||
    !Number.isFinite(perSecond) ||
    perSecond < 0 ||
    perSecond > field.max
  ) {
    return { refused: rateRange(field) };
  }
  return Number.isInteger(perSecond) && perSecond % kib === 0
    ? { value: perSecond }
    : { refused: "In whole KB/s." };
}

function RateControl({ field, busy, invalid, disabled, submit, onEdit }: ControlProps<"rate">) {
  const [unit, setUnit] = useState(() => rateUnitFor(field.value));
  const [amount, setAmount] = useState(() => String(field.value / unit.bytes));
  const zero = amount.trim() === "0" && field.zero !== undefined;
  return (
    <form
      onSubmit={onSubmit(() => {
        submit(parseRate(field, amount, unit));
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
      {zero ? (
        <span className={unitLook}>{field.zero}</span>
      ) : (
        <Segmented
          label={`${field.label} unit`}
          options={rateUnits.map((candidate) => ({ value: candidate.word, label: candidate.word }))}
          value={unit.word}
          disabled={disabled}
          onValueChange={(word) => {
            setUnit(rateUnits.find((candidate) => candidate.word === word) ?? unit);
            onEdit();
          }}
        />
      )}
      <SaveButton
        shown={amount.trim() === "" || Number(amount) * unit.bytes !== field.value}
        busy={busy}
      />
    </form>
  );
}

export const rateKind: Kind<"rate"> = {
  words: (field) => rateAmount(field, field.value),
  hints: (field) => [
    [
      defaultHint(field, rateAmount(field, field.default)),
      rateRange(field),
      ...(field.zero === undefined ? [] : [`0 means ${field.zero.toLowerCase()}.`]),
    ].join(" "),
  ],
  Control: RateControl,
};
