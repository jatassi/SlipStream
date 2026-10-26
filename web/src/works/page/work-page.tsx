import type { WorksContentType, WorksWorkView } from "../../api/gen/model";
import { useImpactSheet } from "../../targets/impact-sheet";
import { GoneUpstream } from "./gone-upstream";
import { Hero } from "./hero";
import { useWorkAmbient } from "./hero-art";
import { useWorkReads } from "./reads";
import { WorkSections } from "./sections";
import { useWorkActions } from "./work-actions";

/** One Work on one scroll: the hero over its backdrop, then its sections. */
export function WorkPage({
  type,
  work,
  onDeleting,
}: {
  type: WorksContentType;
  work: WorksWorkView;
  onDeleting: () => void;
}) {
  const reads = useWorkReads(work);
  const impact = useImpactSheet();
  const actions = useWorkActions({
    type,
    work,
    reads,
    save: impact.save,
    onDeleting,
  });
  useWorkAmbient(type, work, reads.cells.summary.status === "failed");
  return (
    <>
      <Hero
        type={type}
        work={work}
        summary={reads.cells.summary}
        warning={
          work.goneUpstreamAt !== null && <GoneUpstream type={type} onDelete={actions.openDelete} />
        }
        actions={actions.hero}
      />
      <WorkSections
        type={type}
        work={work}
        reads={reads}
        save={impact.save}
        wantAnother={actions.wantAnother}
      />
      {actions.dialogs}
      {impact.sheet}
    </>
  );
}
