import { RoleChips } from "./role-chips";
import type { NodeState } from "./state";
import { NodeStateMark } from "./state-mark";

/** What every page of a Node opens with beside its name: its state, when it was last seen while away, then its Roles. */
export function NodeHeading({
  state,
  seen,
  roles,
}: {
  state: NodeState;
  seen: string | undefined;
  roles: readonly { readonly role: string; readonly words: string }[];
}) {
  return (
    <div className="flex flex-wrap items-center gap-x-4 gap-y-2">
      <p className="flex items-center gap-2">
        <NodeStateMark state={state} />
        {seen !== undefined && (
          <>
            <span aria-hidden="true" className="text-ink-4">
              ·
            </span>
            <span className="text-meta text-ink-3">{seen}</span>
          </>
        )}
      </p>
      <RoleChips roles={roles} />
    </div>
  );
}
