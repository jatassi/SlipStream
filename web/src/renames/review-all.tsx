import { useState } from "react";

import { Button } from "../ui/button";
import { type Opening, PlanSheet } from "./plan-sheet";
import { RenamingChip } from "./renaming-chip";

const everyFile: Opening = { title: "Rename files", scope: { kind: "library" } };

/** The Naming section's way into the whole library's rename plan, with the files still moving. */
export function ReviewAllRenames() {
  const [opening, setOpening] = useState<Opening>();
  return (
    <div className="flex flex-wrap items-center gap-3">
      <Button
        onClick={() => {
          setOpening(everyFile);
        }}
      >
        Review all renames
      </Button>
      <RenamingChip />
      <PlanSheet
        opening={opening}
        onClose={() => {
          setOpening(undefined);
        }}
      />
    </div>
  );
}
