import { useAttempt } from "../../targets/attempt";
import { usePasskeyOffer } from "../-links/passkey-step";
import { StepFooter } from "./frame";
import type { StepBodyProps } from "./screens";

function Offer({ nav }: StepBodyProps) {
  const added = useAttempt(nav.complete);
  const offer = usePasskeyOffer(added.run);
  return (
    <>
      {offer.failure}
      {added.failure}
      <StepFooter nav={nav} primary={offer.add} />
    </>
  );
}

export function PasskeyStep({ step, nav }: StepBodyProps) {
  if (step.state !== "done") {
    return <Offer step={step} nav={nav} />;
  }
  return (
    <>
      <p>An Admin of this SlipStream has a passkey.</p>
      <StepFooter nav={nav} />
    </>
  );
}
