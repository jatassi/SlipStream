import { Data } from "../ui/data";
import { FileField } from "../ui/file-field";
import { ProgressBar } from "../ui/progress";
import type { RestoreFlow, RestoreStep } from "./flow";
import { SignalLine } from "../ui/signal-line";

const backupFile = ".age,application/octet-stream";

export function UploadStep({
  step,
  flow,
}: {
  step: Extract<RestoreStep, { at: "file" | "uploading" }>;
  flow: RestoreFlow;
}) {
  if (step.at === "uploading") {
    return (
      <div className="flex flex-col gap-2">
        <p>
          Uploading <Data>{step.name}</Data>…
        </p>
        <ProgressBar
          label="Uploading the Backup"
          value={Math.floor(step.sent * 100)}
          motion="live"
        />
      </div>
    );
  }
  return (
    <div className="flex flex-col gap-3">
      <FileField label="Backup file" accept={backupFile} onFile={flow.upload} />
      {step.refusal !== undefined && (
        <SignalLine tone="fail" announce>
          {step.refusal}
        </SignalLine>
      )}
    </div>
  );
}
