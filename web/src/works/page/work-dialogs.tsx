import type { WorksWorkView } from "../../api/gen/model";
import type { ImpactChange } from "../../targets/use-impact-save";
import type { Subject } from "../actions/actions";
import { DeleteDialog } from "../actions/delete-dialog";
import { MonitoringDialog } from "../monitoring/monitoring-dialog";
import { EditDialog } from "./edit-dialog";

type WorkDialogsProps = {
  open: "edit" | "delete" | "monitoring" | undefined;
  subject: Subject;
  work: WorksWorkView;
  files: number;
  save: (change: ImpactChange) => Promise<void>;
  onClose: () => void;
  onDeleting: () => void;
};

/** The Work's Edit, Delete or Monitoring dialog while it is open, each mounted fresh as it opens. */
export function WorkDialogs({
  open,
  subject,
  work,
  files,
  save,
  onClose,
  onDeleting,
}: WorkDialogsProps) {
  const changes = (next: boolean) => {
    if (!next) {
      onClose();
    }
  };
  if (open === "edit") {
    return (
      <EditDialog
        subject={subject}
        work={work}
        files={files}
        save={save}
        open
        onOpenChange={changes}
      />
    );
  }
  if (open === "monitoring") {
    return (
      <MonitoringDialog
        type={subject.type}
        subjects={[subject]}
        includeAsides={work.monitorAsides}
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
