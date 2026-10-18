import { useListCutsSuspense } from "../api/gen/query/targets/targets";
import { RankedList, type RankedRow, tiersOf } from "../ui/ranked-list";
import { Segmented } from "../ui/segmented";
import { SignalChip } from "../ui/signal-chip";
import { type EditionsDraft, keysOf, listOf } from "./editions-draft";
import { type EditionsSentence, editionsSentence } from "./editions-sentence";
import { PickMenu, RemoveButton } from "./list-controls";

const kinds = [
  { value: "any", label: "Any cut" },
  { value: "cuts", label: "Choose cuts" },
] as const;

const rowsOf = (keys: readonly string[]): RankedRow<string>[] =>
  keys.map((key) => ({ kind: "item", id: key, value: key, tiedAbove: false }));

function Meaning({ sentence }: { sentence: EditionsSentence }) {
  return (
    <p
      aria-live="polite"
      className={
        sentence.warning
          ? "flex flex-wrap items-center gap-x-3 gap-y-2 rounded-card bg-warn-card px-5 py-3 text-warn-text ring-1 ring-warn-line ring-inset"
          : "text-ink-1"
      }
    >
      {sentence.warning && <SignalChip kind="warn">No Standard</SignalChip>}
      <span className="min-w-0 flex-[1_1_24ch]">{sentence.text}</span>
    </p>
  );
}

function ChosenCuts({
  value,
  onChange,
  nameOf,
  offered,
}: {
  value: EditionsDraft;
  onChange: (next: EditionsDraft) => void;
  nameOf: (key: string) => string;
  offered: readonly { key: string; name: string }[];
}) {
  return (
    <div className="flex flex-col items-start gap-3">
      <div className="w-full">
        <RankedList
          label="Cuts"
          caption="Each listed cut is wanted at every Target. Works show their cuts in this order."
          rows={rowsOf(value.cuts)}
          name={nameOf}
          onChange={(rows) => {
            onChange({ ...value, cuts: keysOf(tiersOf(rows).flat()) });
          }}
          renderRow={(key) => <span className="text-ink-1">{nameOf(key)}</span>}
          renderTrailing={(key) => (
            <RemoveButton
              name={nameOf(key)}
              disabled={value.cuts.length === 1}
              onRemove={() => {
                onChange({ ...value, cuts: keysOf(value.cuts.filter((cut) => cut !== key)) });
              }}
            />
          )}
        />
      </div>
      <PickMenu
        label="Add cut"
        options={offered}
        onPick={(key) => {
          onChange({ ...value, cuts: [...value.cuts, key] });
        }}
      />
    </div>
  );
}

export function EditionsControl({
  value,
  onChange,
  contentType,
}: {
  value: EditionsDraft;
  onChange: (next: EditionsDraft) => void;
  contentType: string;
}) {
  const { cuts } = useListCutsSuspense().data;
  const nameOf = (key: string) => cuts.find((cut) => cut.key === key)?.name ?? key;
  const offered = cuts.filter(
    (cut) =>
      (cut.origin !== "type" || cut.contentType === contentType) && !value.cuts.includes(cut.key),
  );
  return (
    <div className="flex flex-col gap-4">
      <div>
        <Segmented
          label="Editions"
          options={kinds}
          value={value.kind}
          onValueChange={(kind) => {
            onChange({ ...value, kind });
          }}
        />
      </div>
      {value.kind === "cuts" && (
        <ChosenCuts value={value} onChange={onChange} nameOf={nameOf} offered={offered} />
      )}
      <Meaning sentence={editionsSentence(listOf(value, nameOf))} />
    </div>
  );
}
