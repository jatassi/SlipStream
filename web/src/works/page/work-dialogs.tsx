import type { WorksWorkView } from "../../api/gen/model";
import type { ImpactChange } from "../../targets/use-impact-save";
import type { Subject } from "../actions/actions";
import { DeleteDialog } from "../actions/delete-dialog";
import { EditDialog } from "./edit-dialog";

/** The Work's Edit or Delete dialog while it is open, each mounted fresh as it opens. */
export function WorkDialogs({
  open,
  subject,
  work,
  files,
  save,
  onClose,
  onDeleting,
}: {
  open: "edit" | "delete" | undefined;
  subject: Subject;
  work: WorksWorkView;
  files: number;
  save: (change: ImpactChange) => Promise<void>;
  onClose: () => void;
  onDeleting: () => void;
}) {
  const changes = (next: boolean) => {
    if (!next) {
      onClose();
    }
  };
  if (open === "edit") {
    return (
      <EditDialog
        subject={subject}
        profileId={work.profile.id}
        rootId={work.root?.id}
        files={files}
        save={save}
        open
        onOpenChange={changes}
      />
    );
  }
  if (open === "delete") {
    return (
      <DeleteDialog subjects={[subject]} open onOpenChange={changes} onDeleting={onDeleting} />
    );
  }
  return null;
}
