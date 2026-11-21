import type { RestoreSummary } from "../api/gen/model";
import { Checkbox } from "../ui/checkbox";
import { oldControllerOffLabel, oldControllerSentence } from "./words";

export type Ticks = {
  readonly removed: ReadonlySet<string>;
  readonly toggle: (id: string, kept: boolean) => void;
};

function NodeTicks({ summary, removed, toggle }: Ticks & { summary: RestoreSummary }) {
  const workers = (summary.nodes ?? []).slice(1);
  if (workers.length === 0) {
    return null;
  }
  return (
    <fieldset className="flex flex-col gap-2">
      <legend className="mb-1 font-semibold text-ink-2">Nodes to keep</legend>
      <p className="text-meta text-ink-3">Untick each Node you removed after this Backup was taken.</p>
      {workers.map((node) => (
        <Checkbox
          key={node.id}
          label={node.name}
          checked={!removed.has(node.id)}
          onCheckedChange={(kept) => {
            toggle(node.id, kept);
          }}
        />
      ))}
    </fieldset>
  );
}

export function NewMachine({
  summary,
  off,
  setOff,
  ...ticks
}: Ticks & { summary: RestoreSummary; off: boolean; setOff: (off: boolean) => void }) {
  return (
    <>
      <p>{oldControllerSentence(summary.controller)}</p>
      <Checkbox
        label={oldControllerOffLabel(summary.controller)}
        checked={off}
        onCheckedChange={setOff}
      />
      <NodeTicks summary={summary} {...ticks} />
    </>
  );
}
