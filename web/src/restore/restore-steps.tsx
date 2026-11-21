import type { ReactNode } from "react";

import type { RestoreFlow } from "./flow";
import { PassphraseStep } from "./passphrase-step";
import { RestartingStep } from "./restarting-step";
import { SummaryStep } from "./summary-step";
import { UploadStep } from "./upload-step";

/**
 * The restore at whichever step it stands. `lead` opens the file step, and
 * `signIn` runs once the restarted Node responds.
 */
export function RestoreSteps({
  flow,
  lead,
  onCancel,
  signIn,
  children,
}: {
  flow: RestoreFlow;
  lead: string;
  onCancel: () => void;
  signIn: () => void;
  children?: ReactNode;
}) {
  const { step } = flow;
  if (step.at === "restarting") {
    return <RestartingStep before={step.before} onBack={signIn} />;
  }
  if (step.at === "summary") {
    return <SummaryStep step={step} flow={flow} onCancel={onCancel} />;
  }
  if (step.at === "passphrase") {
    return <PassphraseStep step={step} flow={flow} />;
  }
  return (
    <div className="flex flex-col gap-4">
      <p>{lead}</p>
      <UploadStep step={step} flow={flow} />
      {step.at === "file" && children}
    </div>
  );
}
