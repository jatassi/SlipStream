import type { TargetsEstimate, TargetsTarget } from "../api/gen/model";
import { useGetProfileSuspense, useListTargetsSuspense } from "../api/gen/query/targets/targets";
import { RankedList, type RankedRow, tiersOf } from "../ui/ranked-list";
import type { SectionProps } from "./editor";
import { PickMenu, RemoveButton } from "./list-controls";
import {
  type EntryDraft,
  type ProfileDraft,
  downloaded,
  sourceLost,
  sourcesAbove,
} from "./profile-draft";
import { TargetRow, containersOf } from "./profile-target-row";

type Estimates = ReadonlyMap<string, TargetsEstimate>;

const rowsOf = (entries: readonly EntryDraft[]): RankedRow<EntryDraft>[] =>
  entries.map((entry) => ({ kind: "item", id: entry.targetId, value: entry, tiedAbove: false }));

function WantsNothing() {
  return (
    <p className="rounded-card bg-glass px-4 py-3 text-ink-2 ring-1 ring-hairline ring-inset">
      This Profile has no Targets. Add one to choose which versions its Works keep.
    </p>
  );
}

function AddTarget({ held, onPick }: { held: readonly string[]; onPick: (id: string) => void }) {
  const { targets } = useListTargetsSuspense().data;
  return (
    <PickMenu
      label="Add Target"
      options={targets.flatMap((target) =>
        held.includes(target.id) ? [] : [{ key: target.id, name: target.name }],
      )}
      onPick={onPick}
    />
  );
}

type Ranks = {
  readonly entries: readonly EntryDraft[];
  readonly contentType: string;
  readonly estimates: Estimates;
  readonly onSet: (entries: readonly EntryDraft[]) => void;
};

function EntryRow({
  row,
  byId,
  ranks: { entries, contentType, estimates, onSet },
}: {
  row: EntryDraft;
  byId: ReadonlyMap<string, TargetsTarget>;
  ranks: Ranks;
}) {
  const index = entries.indexOf(row);
  return (
    <TargetRow
      name={byId.get(row.targetId)?.name ?? row.targetId}
      target={byId.get(row.targetId)}
      contentType={contentType}
      estimate={estimates.get(row.targetId)}
      containers={containersOf(
        row.targetId,
        entries.map((entry) => entry.targetId),
        byId,
      )}
      sourcing={{
        entry: row,
        sources: sourcesAbove(entries, index),
        lost: sourceLost(entries, index),
        nameOf: (id) => byId.get(id)?.name ?? id,
        onEntry: (entry) => {
          onSet(entries.map((other) => (other === row ? entry : other)));
        },
      }}
    />
  );
}

function TargetRanks(ranks: Ranks) {
  const { entries, onSet } = ranks;
  const { targets } = useListTargetsSuspense().data;
  const byId = new Map(targets.map((target) => [target.id, target]));
  const nameOf = (id: string) => byId.get(id)?.name ?? id;
  return (
    <RankedList
      label="Targets"
      caption="Each Work wants a file at every Target. A file is named after the first Target it fills."
      rows={rowsOf(entries)}
      name={(entry) => nameOf(entry.targetId)}
      onChange={(next) => {
        onSet(tiersOf(next).flat());
      }}
      renderRow={(row) => <EntryRow row={row} byId={byId} ranks={ranks} />}
      renderTrailing={(row) => (
        <RemoveButton
          name={nameOf(row.targetId)}
          onRemove={() => {
            onSet(entries.filter((other) => other !== row));
          }}
        />
      )}
    />
  );
}

function TargetList({
  draft,
  onChange,
  estimates,
}: SectionProps<ProfileDraft> & { estimates: Estimates }) {
  const set = (targets: readonly EntryDraft[]) => {
    onChange({ ...draft, targets });
  };
  const held = draft.targets.map((entry) => entry.targetId);
  return (
    <div className="flex flex-col items-start gap-3">
      <div className="w-full">
        {draft.targets.length === 0 ? (
          <WantsNothing />
        ) : (
          <TargetRanks
            entries={draft.targets}
            contentType={draft.contentType}
            estimates={estimates}
            onSet={set}
          />
        )}
      </div>
      <AddTarget
        held={held}
        onPick={(id) => {
          set([...draft.targets, downloaded(id)]);
        }}
      />
    </div>
  );
}

function SavedTargetList({ id, ...props }: SectionProps<ProfileDraft> & { id: string }) {
  const { estimates } = useGetProfileSuspense(id).data;
  return (
    <TargetList
      {...props}
      estimates={new Map(estimates.map((estimate) => [estimate.targetId, estimate]))}
    />
  );
}

export function ProfileTargetsSection(props: SectionProps<ProfileDraft>) {
  return props.draft.id === null ? (
    <TargetList {...props} estimates={new Map()} />
  ) : (
    <SavedTargetList {...props} id={props.draft.id} />
  );
}
