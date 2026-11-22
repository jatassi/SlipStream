import type { WorksWorkView } from "../../api/gen/model";
import { PlanSheet } from "../../renames/plan-sheet";
import type { ImpactChange } from "../../targets/use-impact-save";
import type { Subject } from "../actions/actions";
import { MoveDialog } from "../../moves/move-dialog";
import { DeleteDialog } from "../actions/delete-dialog";
import { MonitoringDialog } from "../monitoring/monitoring-dialog";
import { EditDialog } from "./edit-dialog";

type WorkDialogsProps = {
  open: "edit" | "delete" | "monitoring" | "rename" | "move" | undefined;
  subject: Subject;
  work: WorksWorkView;
  files: number;
  save: (change: ImpactChange) => Promise<void>;
  onClose: () => void;
  onRename: () => void;
  onMove: () => void;
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

function WorkMove({
  subject,
  work,
  onClose,
}: {
  subject: Subject;
  work: WorksWorkView;
  onClose: () => void;
}) {
  return (
    <MoveDialog
      type={subject.type}
      works={[subject]}
      {...(work.root !== null && { from: work.root.id })}
      origin="work"
      open
      onOpenChange={onClose}
    />
  );
}

/** The Work's Edit, Move, Delete or Monitoring dialog or its rename plan while it is open, each mounted fresh as it opens. */
export function WorkDialogs(props: WorkDialogsProps) {
  const { open, subject, work, onClose, onDeleting } = props;
  const changes = (next: boolean) => {
    if (!next) {
      onClose();
    }
  };
  if (open === "edit") {
    return <EditDialog {...props} open onOpenChange={changes} />;
  }
  if (open === "move") {
    return <WorkMove subject={subject} work={work} onClose={onClose} />;
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
