import { useState } from "react";

import { assignmentOf } from "../../../../nodes/roles";
import { Actions, Suggestion } from "./actions";
import { RoleGroups } from "./groups";
import type { Change } from "./plan";
import { ReviewDialog } from "./review";
import { type Roles, useRoles } from "./use-roles";
import { WriteOffDialog } from "./write-off";

type Opened =
  | { readonly dialog: "none" }
  | { readonly dialog: "review"; readonly change: Change }
  | { readonly dialog: "write-off" };

const closed: Opened = { dialog: "none" };

function Dialogs({
  opened,
  roles: { board, seat, setDraft, reload },
  onClose,
}: {
  opened: Opened;
  roles: Roles;
  onClose: () => void;
}) {
  if (opened.dialog === "none") {
    return null;
  }
  if (opened.dialog === "write-off") {
    return <WriteOffDialog board={board} seat={seat} onClose={onClose} onWrittenOff={reload} />;
  }
  return (
    <ReviewDialog
      change={opened.change}
      onClose={onClose}
      onApplied={async () => {
        await reload();
        setDraft(undefined);
        onClose();
      }}
    />
  );
}

/**
 * A Node's Roles, by Role Group. A change goes through the Review, and the
 * work the Node holds can be written off.
 */
export function RolesSection({ nodeId }: { nodeId: string }) {
  const roles = useRoles(nodeId);
  const { board, seat, draft, changed, setDraft } = roles;
  const [opened, setOpened] = useState<Opened>(closed);
  return (
    <div className="flex flex-col gap-4">
      {seat.assigned.length === 0 && seat.suggested.length > 0 && !changed && (
        <Suggestion
          board={board}
          seat={seat}
          onTick={() => {
            setDraft(new Set([...draft, ...seat.suggested]));
          }}
        />
      )}
      <RoleGroups {...roles} onDraft={setDraft} />
      <Actions
        seat={seat}
        changed={changed}
        onReview={() => {
          setOpened({
            dialog: "review",
            change: { basis: board.basis, set: [assignmentOf(seat.id, draft)] },
          });
        }}
        onDiscard={() => {
          setDraft(undefined);
        }}
        onWriteOff={() => {
          setOpened({ dialog: "write-off" });
        }}
      />
      <Dialogs
        opened={opened}
        roles={roles}
        onClose={() => {
          setOpened(closed);
        }}
      />
    </div>
  );
}
