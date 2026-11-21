import { AlertLine } from "../ui/alert-line";
import { Data } from "../ui/data";
import { FileField } from "../ui/file-field";
import { LiveBar } from "../ui/live-bar";
import type { RestoreFlow, RestoreStep } from "./flow";

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
        <LiveBar label="Uploading the Backup" value={Math.round(step.sent * 100)} />
      </div>
    );
  }
  return (
    <div className="flex flex-col gap-3">
      <FileField label="Backup file" accept={backupFile} onFile={flow.upload} />
      {step.refusal !== undefined && <AlertLine text={step.refusal} />}
    </div>
  );
}
