import { useState } from "react";

import type { WorksContentType } from "../../api/gen/model";
import { MoveDialog } from "../../moves/move-dialog";
import { Button } from "../../ui/button";
import type { Rooted } from "./root-dialog";

/** The Works with files, which a Move copies to the new root instead. */
export function WithFiles({
  type,
  staying,
}: {
  type: WorksContentType;
  staying: readonly Rooted[];
}) {
  const [moving, setMoving] = useState(false);
  const [only] = staying;
  const many = type.work.many.toLowerCase();
  return (
    <div className="flex flex-col items-start gap-2">
      <p>
        {staying.length === 1 && only !== undefined
          ? `${only.label} has files. Moving it copies its files to the new root.`
          : `${staying.length} of these ${many} have files. Moving them copies their files to the new root.`}
      </p>
      <Button
        onClick={() => {
          setMoving(true);
        }}
      >
        Move {staying.length === 1 ? "it" : "them"}…
      </Button>
      {moving && (
        <MoveDialog type={type} works={staying} origin="bulk" open onOpenChange={setMoving} />
      )}
    </div>
  );
}
