import type { WorksWorkView } from "../../api/gen/model";
import { PlanSheet } from "../../renames/plan-sheet";
import type { ImpactChange } from "../../targets/use-impact-save";
import type { Subject } from "../actions/actions";
import { DeleteDialog } from "../actions/delete-dialog";
import { MonitoringDialog } from "../monitoring/monitoring-dialog";
import { EditDialog } from "./edit-dialog";

type WorkDialogsProps = {
  open: "edit" | "delete" | "monitoring" | "rename" | undefined;
  subject: Subject;
  work: WorksWorkView;
  files: number;
  save: (change: ImpactChange) => Promise<void>;
  onClose: () => void;
  onRename: () => void;
  onDeleting: () => void;
};

function WorkPlan({ work, onClose }: { work: WorksWorkView; onClose: () => void }) {
  return (
    <PlanSheet
      opening={{ title: `Rename files of ${work.label}`, scope: { kind: "work", id: work.id } }}
      onClose={onClose}
    />
  );
}

/** The Work's Edit, Delete or Monitoring dialog or its rename plan while it is open, each mounted fresh as it opens. */
export function WorkDialogs({
  open,
  subject,
  work,
  files,
  save,
  onClose,
  onRename,
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
        onRename={onRename}
      />
    );
  }
  if (open === "rename") {
    return <WorkPlan work={work} onClose={onClose} />;
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
