import { useNavigate, useRouter } from "@tanstack/react-router";
import type { ReactNode } from "react";

import {
  useGetRoleBoardSuspense,
  useListNodesSuspense,
} from "../../../../api/gen/query/nodes/nodes";
import { useRefreshNodes } from "../-node";
import { WriteOffDialog } from "../-roles/write-off";
import { ReachDialog } from "./reach";
import { RemoveDialog } from "./remove";
import { RenameDialog } from "./rename";

/** What the Admin asked to do to one Node from its Roster row or its Overview. */
export type Acting = {
  readonly kind: "rename" | "write-off" | "remove" | "reach";
  readonly nodeId: string;
};

/** The dialog for what the Admin is doing to a Node, and the reload once it is done. */
export function NodeDialogs({
  acting,
  onClose,
}: {
  acting: Acting | undefined;
  onClose: () => void;
}) {
  const board = useGetRoleBoardSuspense().data;
  const { nodes } = useListNodesSuspense().data;
  const refreshNodes = useRefreshNodes();
  const router = useRouter();
  const navigate = useNavigate();
  const node = nodes.find(({ id }) => id === acting?.nodeId);
  const seat = board.nodes.find(({ id }) => id === acting?.nodeId);
  if (acting === undefined || node === undefined || seat === undefined) {
    return null;
  }
  const done = async () => {
    await refreshNodes();
    await router.invalidate();
    onClose();
  };
  const controller = nodes.find(({ self }) => self)?.name ?? "the Controller";
  const dialogs: { readonly [K in Acting["kind"]]: () => ReactNode } = {
    rename: () => (
      <RenameDialog nodeId={node.id} name={node.name} onClose={onClose} onRenamed={done} />
    ),
    "write-off": () => (
      <WriteOffDialog board={board} seat={seat} onClose={onClose} onWrittenOff={done} />
    ),
    reach: () => (
      <ReachDialog node={node} controller={controller} onClose={onClose} onChanged={done} />
    ),
    remove: () => (
      <RemoveDialog
        board={board}
        seat={seat}
        onClose={onClose}
        onRemoved={async () => {
          await navigate({ to: "/nodes", search: {} });
          await refreshNodes();
        }}
      />
    ),
  };
  return dialogs[acting.kind]();
}
