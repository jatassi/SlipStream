import { AddressForm } from "../../settings/address";
import { SubmitButton } from "../../ui/form";
import { StepFooter } from "./frame";
import type { StepBodyProps } from "./screens";

export function AddressStep({ nav }: StepBodyProps) {
  return (
    <AddressForm onSaved={nav.complete}>
      <StepFooter nav={nav} primary={<SubmitButton label="Continue" />} />
    </AddressForm>
  );
}
