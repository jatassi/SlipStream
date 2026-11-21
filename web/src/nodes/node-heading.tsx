import { RelativeTime } from "../ui/relative-time";
import { RoleChips } from "./role-chips";
import type { NodeState } from "./state";
import { NodeStateMark } from "./state-mark";

/** What every page of a Node opens with beside its name: its state, when it was last seen while away, then its Roles. */
export function NodeHeading({
  state,
  seenAt,
  roles,
}: {
  state: NodeState;
  /** When it was last seen, while it is away. */
  seenAt: string | undefined;
  roles: readonly { readonly role: string; readonly words: string }[];
}) {
  return (
    <div className="flex flex-wrap items-center gap-x-4 gap-y-2">
      <p className="flex items-center gap-2">
        <NodeStateMark state={state} />
        {seenAt !== undefined && (
          <>
            <span aria-hidden="true" className="text-ink-4">
              ·
            </span>
            <span className="text-meta text-ink-3">
              Last seen <RelativeTime at={seenAt} form="age" />
            </span>
          </>
        )}
      </p>
      <RoleChips roles={roles} />
    </div>
  );
}
