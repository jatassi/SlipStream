import type { WorksContentType } from "../api/gen/model";
import { DeleteDialog } from "../works/actions/delete-dialog";
import { ProfileDialog } from "../works/actions/profile-dialog";
import { RootDialog } from "../works/actions/root-dialog";
import type { LibraryWork } from "./rows";

export type BulkDialogKind = "profile" | "root" | "delete";

/** The open bulk dialog, drawn fresh each time it opens so it starts from the defaults. */
export function BulkDialog({
  kind,
  type,
  picked,
  onClose,
}: {
  kind: BulkDialogKind | undefined;
  /** The one Content Type of the picks, which Profiles and roots belong to. */
  type: WorksContentType | undefined;
  picked: readonly LibraryWork[];
  onClose: () => void;
}) {
  const close = (open: boolean) => {
    if (!open) {
      onClose();
    }
  };
  if (kind === "delete") {
    return <DeleteDialog subjects={picked} open onOpenChange={close} />;
  }
  if (type === undefined || kind === undefined) {
    return null;
  }
  return kind === "profile" ? (
    <ProfileDialog type={type} subjects={picked} profileId={undefined} open onOpenChange={close} />
  ) : (
    <RootDialog type={type} works={picked} open onOpenChange={close} />
  );
}
