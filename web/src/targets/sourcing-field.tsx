import { countWords } from "../format";
import { Segmented } from "../ui/segmented";
import { Select } from "../ui/select";
import { SignalLine } from "../ui/signal-line";
import { DraftField } from "./field";
import {
  type EntryDraft,
  type GraceDraft,
  type GraceUnit,
  type Made,
  graceProblem,
} from "./profile-draft";

const graceUnits: readonly { value: GraceUnit; label: string }[] = [
  { value: "hours", label: "hours" },
  { value: "days", label: "days" },
];

function graceWords(grace: GraceDraft): string {
  if (graceProblem(grace) !== undefined) {
    return "a wait";
  }
  const count = Number(grace.amount);
  return grace.unit === "days"
    ? countWords(count, { one: "day", many: "days" })
    : countWords(count, { one: "hour", many: "hours" });
}

type Key = "download" | `${Made}:${string}`;

const keyOf = (entry: EntryDraft): Key =>
  entry.sourcing === "download" ? "download" : `${entry.sourcing}:${entry.from}`;

function GraceField({
  grace,
  from,
  onGrace,
}: {
  grace: GraceDraft;
  from: string;
  onGrace: (grace: GraceDraft) => void;
}) {
  return (
    <div className="flex flex-wrap items-start gap-3">
      <div className="w-60">
        <DraftField
          label="Wait for a download for"
          value={grace.amount}
          inputMode="decimal"
          error={graceProblem(grace)}
          description={`Counted from the import of the ${from} file.`}
          onChange={(amount) => {
            onGrace({ ...grace, amount });
          }}
        />
      </div>
      <div className="pt-7">
        <Segmented
          label="Wait unit"
          options={graceUnits}
          value={grace.unit}
          onValueChange={(unit) => {
            onGrace({ ...grace, unit });
          }}
        />
      </div>
    </div>
  );
}

type Choice = { readonly label: string; readonly entry: EntryDraft };

function choicesOf(
  entry: EntryDraft,
  offered: readonly string[],
  nameOf: (id: string) => string,
): Choice[] {
  const { targetId, grace } = entry;
  return [
    { label: "Downloaded", entry: { targetId, sourcing: "download", grace } },
    ...offered.flatMap((from): Choice[] => [
      { label: `Made from ${nameOf(from)}`, entry: { targetId, sourcing: "derive", from, grace } },
      {
        label: `Downloaded, or made from ${nameOf(from)} after ${graceWords(grace)}`,
        entry: { targetId, sourcing: "download_then_derive", from, grace },
      },
    ]),
  ];
}

export type Sourcing = {
  readonly entry: EntryDraft;
  /** The downloaded Targets above the row, which it can be made from. */
  readonly sources: readonly string[];
  readonly lost: boolean;
  readonly nameOf: (id: string) => string;
  readonly onEntry: (entry: EntryDraft) => void;
};

/**
 * How a Target with Conversion settings gets its file: downloaded, or made
 * from a downloaded Target above it, at once or after a wait for a download.
 */
export function SourcingField({
  name,
  sourcing: { entry, sources, lost, nameOf, onEntry },
}: {
  name: string;
  sourcing: Sourcing;
}) {
  const offered = lost && entry.sourcing !== "download" ? [...sources, entry.from] : sources;
  const choices = choicesOf(entry, offered, nameOf);
  return (
    <div className="flex flex-col gap-3 pt-2">
      <div className="w-full max-w-md">
        <Select
          label={`How ${name} gets its file`}
          options={choices.map((choice) => ({ value: keyOf(choice.entry), label: choice.label }))}
          value={keyOf(entry)}
          onValueChange={(value) => {
            const chosen = choices.find((choice) => keyOf(choice.entry) === value);
            if (chosen !== undefined) {
              onEntry(chosen.entry);
            }
          }}
        />
      </div>
      {lost && entry.sourcing !== "download" && (
        <SignalLine tone="fail">
          {`${nameOf(entry.from)} isn't a downloaded Target above ${name}. Move it up, or choose another source.`}
        </SignalLine>
      )}
      {entry.sourcing === "download_then_derive" && (
        <GraceField
          grace={entry.grace}
          from={nameOf(entry.from)}
          onGrace={(next) => {
            onEntry({ ...entry, grace: next });
          }}
        />
      )}
    </div>
  );
}
