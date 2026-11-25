import { OpenSlipStream } from "../routes/-links/open-slipstream";
import { PasskeyStep as AddPasskey } from "../routes/-links/passkey-step";
import { useAccount } from "../session/session";
import type { WelcomeStepProps } from "./steps";

export function PasskeyStep({ next }: WelcomeStepProps) {
  return <AddPasskey onDone={next} />;
}

export function DoneStep() {
  const { displayName } = useAccount();
  return <OpenSlipStream sentence={`Your Account is ready, ${displayName}.`} to="/portal" />;
}
