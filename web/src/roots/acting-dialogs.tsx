import type { RootsRoot } from "../api/gen/model";
import { DeleteRootDialog } from "./delete-root";
import { MarkGoneDialog } from "./mark-gone";
import { MoveRootChooser, MoveRootWorks } from "./move-root-chooser";
import { LibraryRootDialog } from "./root-dialog";

/** The dialog a root's row, or an Add button, opens. */
export type Acting =
  | { readonly kind: "add"; readonly contentType: string }
  | { readonly kind: "edit" | "delete" | "move" | "move-works"; readonly rootId: string }
  | { readonly kind: "gone"; readonly rootId: string; readonly name: string };

export function ActingDialogs({
  acting,
  roots,
  onAct,
}: {
  acting: Acting | undefined;
  roots: readonly RootsRoot[];
  /** Opens the next dialog, or closes them with undefined. */
  onAct: (acting: Acting | undefined) => void;
}) {
  if (acting === undefined) {
    return null;
  }
  const close = (open: boolean) => {
    if (!open) {
      onAct(undefined);
    }
  };
  if (acting.kind === "add") {
    return <LibraryRootDialog open onOpenChange={close} contentType={acting.contentType} />;
  }
  const root = roots.find((candidate) => candidate.id === acting.rootId);
  if (root === undefined) {
    return null;
  }
  if (acting.kind === "gone") {
    return <MarkGoneDialog root={root} name={acting.name} onOpenChange={close} />;
  }
  if (acting.kind === "move") {
    return <MoveRootChooser root={root} onAct={onAct} onOpenChange={close} />;
  }
  if (acting.kind === "move-works") {
    return <MoveRootWorks root={root} onOpenChange={close} />;
  }
  return acting.kind === "edit" ? (
    <LibraryRootDialog open onOpenChange={close} root={root} />
  ) : (
    <DeleteRootDialog open onOpenChange={close} root={root} />
  );
}
