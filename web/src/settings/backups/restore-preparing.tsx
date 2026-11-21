import type { BackupsDestination } from "../../api/gen/model";
import { AlertLine } from "../../ui/alert-line";
import { Button } from "../../ui/button";
import { Spinner } from "../../ui/spinner";
import type { Prepare } from "./restore-prepare";
import type { Names } from "./words";

export function Preparing({
  prepare,
  pull,
  names,
  onCancel,
}: {
  prepare: Exclude<Prepare, { at: "staged" }>;
  pull: (from: BackupsDestination) => Promise<void>;
  names: Names;
  onCancel: () => void;
}) {
  if (prepare.at === "refused") {
    return (
      <div className="flex flex-col items-start gap-3">
        <AlertLine text={prepare.refusal} />
        <Button onClick={onCancel}>Close</Button>
      </div>
    );
  }
  if (prepare.at === "pull") {
    const node = names(prepare.from.nodeId);
    return (
      <div className="flex flex-col items-start gap-3">
        <p>{`This Backup's file isn't on this machine. SlipStream copies it back from ${node} first.`}</p>
        <div className="flex flex-wrap gap-2">
          <Button look="primary" onClick={() => void pull(prepare.from)}>
            Copy it back
          </Button>
          <Button look="ghost" onClick={onCancel}>
            Cancel
          </Button>
        </div>
      </div>
    );
  }
  return (
    <output className="flex items-center gap-2 text-ink-1">
      <Spinner className="size-4 text-ink-2" />
      {prepare.at === "pulling"
        ? `Copying the Backup back from ${names(prepare.from.nodeId)}…`
        : "Preparing the restore…"}
    </output>
  );
}
