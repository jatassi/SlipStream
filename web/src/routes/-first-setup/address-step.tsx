import { AddressForm } from "../../settings/address";
import { SubmitButton } from "../../ui/form";
import { StepFooter } from "./frame";
import type { StepBodyProps } from "./screens";

export function AddressStep({ step, nav }: StepBodyProps) {
  return (
    <AddressForm
      onSaved={nav.complete}
      onUnchanged={step.state === "done" ? nav.complete : undefined}
    >
      <StepFooter nav={nav} primary={<SubmitButton label="Continue" />} />
    </AddressForm>
  );
}
