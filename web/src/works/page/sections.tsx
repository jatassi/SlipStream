import type { ReactElement, ReactNode } from "react";

import type { WorksContentType, WorksWorkView } from "../../api/gen/model";
import { Searches } from "../../grabs/searches";
import { Grabs } from "../../grabs/work-sections";
import type { ImpactChange } from "../../targets/use-impact-save";
import { About } from "./about";
import { FilesSection } from "./files-section";
import { MatrixSection } from "./matrix";
import { NeedsYou } from "./needs-you";
import type { WorkReads } from "./reads";

/** The Work page under its hero: Needs you, the matrix, its files, About, then its Grabs and searches. */
export function WorkSections({
  type,
  work,
  reads: { cells, matrix, wants, listing, grabs, cutName, targetName },
  save,
  wantAnother,
}: {
  type: WorksContentType;
  work: WorksWorkView;
  reads: WorkReads;
  save: (change: ImpactChange) => Promise<void>;
  wantAnother: (trigger: ReactElement) => ReactNode;
}) {
  return (
    <div className="mt-7">
      <NeedsYou naming={{ type, work, targetName, cutName }} needs={cells.needsYou} grabs={grabs} />
      <MatrixSection
        type={type}
        matrix={matrix}
        shown={{
          work,
          wants,
          grabs,
          files: listing.files,
          cutName,
          targetName,
          save,
        }}
        wantAnother={wantAnother}
      />
      <FilesSection listing={listing} root={work.root} names={{ cutName, targetName }} />
      <About type={type} work={work} />
      <Grabs workId={work.id} work={work.label} />
      <Searches workId={work.id} />
    </div>
  );
}
