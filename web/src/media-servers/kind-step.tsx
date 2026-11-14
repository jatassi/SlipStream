import { useState } from "react";

import type { MediaserversKindView as Kind } from "../api/gen/model";
import { Button } from "../ui/button";
import { RadioGroup } from "../ui/radio";

/** The Add sheet's first step: which kind of server to add. */
export function KindStep({
  kinds,
  onNext,
}: {
  kinds: readonly Kind[];
  onNext: (kind: Kind) => void;
}) {
  const [first] = kinds;
  if (first === undefined) {
    throw new Error("the kinds list has no kind this sheet adds");
  }
  const [chosen, setChosen] = useState(first);
  return (
    <div className="flex flex-col gap-4">
      <RadioGroup
        label="Media Server"
        options={kinds.map((kind) => ({ value: kind.kind, label: kind.label }))}
        value={chosen.kind}
        onValueChange={(next) => {
          setChosen(kinds.find((kind) => kind.kind === next) ?? first);
        }}
      />
      <div className="flex justify-end">
        <Button
          look="primary"
          onClick={() => {
            onNext(chosen);
          }}
        >
          Next
        </Button>
      </div>
    </div>
  );
}
