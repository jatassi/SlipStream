import type { ReactElement, ReactNode } from "react";

import type { WorksContentType, WorksWorkView } from "../../api/gen/model";
import { Searches } from "../../grabs/searches";
import { Grabs } from "../../grabs/work-sections";
import { WorkHistory } from "../../history/work-history";
import type { ImpactChange } from "../../targets/use-impact-save";
import { About } from "./about";
import { FilesSection } from "./files-section";
import { LevelsSection } from "../levels/levels-section";
import { MatrixSection } from "./matrix";
// oxlint-disable-next-line import/max-dependencies -- the page under its hero is every section in order
import { NeedsYou } from "./needs-you";
import type { WorkReads } from "./reads";

/** The Work page under its hero: Needs you, the matrix or the levels, its files, About, then its Grabs, searches and History. */
export function WorkSections({
  type,
  work,
  reads,
  save,
  wantAnother,
}: {
  type: WorksContentType;
  work: WorksWorkView;
  reads: WorkReads;
  save: (change: ImpactChange) => Promise<void>;
  wantAnother: (trigger: ReactElement) => ReactNode;
}) {
  const { cells, matrix, wants, listing, grabs, searching, cutName, targetName } = reads;
  const shown = { work, wants, grabs, files: listing.files, cutName, targetName, save };
  return (
    <div className="mt-7">
      <NeedsYou
        naming={{ type, work, targetName, cutName }}
        needs={cells.needsYou}
        grabs={grabs}
        stops={searching.stops}
      />
      {type.items === "many" ? (
        <LevelsSection type={type} work={work} reads={reads} shown={shown} />
      ) : (
        <MatrixSection type={type} matrix={matrix} shown={shown} wantAnother={wantAnother} />
      )}
      <FilesSection
        listing={listing}
        root={work.root}
        names={{ cutName, targetName }}
        workId={work.id}
        noun={type.work.many}
      />
      <About type={type} work={work} statuses={cells.items} />
      <Grabs workId={work.id} work={work.label} />
      <Searches work={work} />
      <WorkHistory type={type} workId={work.id} />
    </div>
  );
}
