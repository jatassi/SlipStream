import type { TargetsEstimate } from "../api/gen/model";
import { useGetProfileSuspense, useListTargetsSuspense } from "../api/gen/query/targets/targets";
import { RankedList, type RankedRow, tiersOf } from "../ui/ranked-list";
import type { SectionProps } from "./editor";
import { PickMenu, RemoveButton } from "./list-controls";
import type { ProfileDraft } from "./profile-draft";
import { TargetRow, containersOf } from "./profile-target-row";

type Estimates = ReadonlyMap<string, TargetsEstimate>;

const rowsOf = (ids: readonly string[]): RankedRow<string>[] =>
  ids.map((id) => ({ kind: "item", id, value: id, tiedAbove: false }));

function WantsNothing() {
  return (
    <p className="rounded-card bg-glass px-4 py-3 text-ink-2 ring-1 ring-hairline ring-inset">
      This Profile wants nothing. Add a Target to say which versions its Works keep.
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

function TargetRanks({
  ids,
  estimates,
  onSet,
}: {
  ids: readonly string[];
  estimates: Estimates;
  onSet: (ids: readonly string[]) => void;
}) {
  const { targets } = useListTargetsSuspense().data;
  const byId = new Map(targets.map((target) => [target.id, target]));
  const nameOf = (id: string) => byId.get(id)?.name ?? id;
  return (
    <RankedList
      label="Targets"
      caption="Each Work wants a file at every Target. A file is named after the first Target it fills."
      rows={rowsOf(ids)}
      name={nameOf}
      onChange={(next) => {
        onSet(tiersOf(next).flat());
      }}
      renderRow={(id) => (
        <TargetRow
          name={nameOf(id)}
          estimate={estimates.get(id)}
          containers={containersOf(id, ids, byId)}
        />
      )}
      renderTrailing={(id) => (
        <RemoveButton
          name={nameOf(id)}
          onRemove={() => {
            onSet(ids.filter((other) => other !== id));
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
  const set = (ids: readonly string[]) => {
    onChange({ ...draft, targets: ids });
  };
  return (
    <div className="flex flex-col items-start gap-3">
      <div className="w-full">
        {draft.targets.length === 0 ? (
          <WantsNothing />
        ) : (
          <TargetRanks ids={draft.targets} estimates={estimates} onSet={set} />
        )}
      </div>
      <AddTarget
        held={draft.targets}
        onPick={(id) => {
          set([...draft.targets, id]);
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
