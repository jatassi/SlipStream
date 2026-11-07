import type { WorksContentType } from "../api/gen/model";
import { DeleteDialog } from "../works/actions/delete-dialog";
import { ProfileDialog } from "../works/actions/profile-dialog";
import { RootDialog } from "../works/actions/root-dialog";
import { MonitoringDialog } from "../works/monitoring/monitoring-dialog";
import type { LibraryWork } from "./rows";

export type BulkDialogKind = "profile" | "root" | "monitoring" | "delete";

/** The open bulk dialog, drawn fresh each time it opens so it starts from the defaults. */
export function BulkDialog({
  kind,
  type,
  picked,
  onClose,
}: {
  kind: BulkDialogKind | undefined;
  /** The one Content Type of the picks, which Profiles, roots and presets belong to. */
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
  if (kind === "profile") {
    return (
      <ProfileDialog
        type={type}
        subjects={picked}
        profileId={undefined}
        open
        onOpenChange={close}
      />
    );
  }
  if (kind === "root") {
    return <RootDialog type={type} works={picked} open onOpenChange={close} />;
  }
  return (
    <MonitoringDialog
      type={type}
      subjects={picked}
      includeAsides={false}
      open
      onOpenChange={close}
    />
  );
}
