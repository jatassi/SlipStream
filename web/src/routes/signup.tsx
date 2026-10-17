import { createFileRoute } from "@tanstack/react-router";

import { requireEntry } from "../session/gate";
import { usePasskeySupport } from "../session/passkey";
import { type LiveProps, LinkPage } from "./-links/link-page";
import { PasskeyStep } from "./-links/passkey-step";
import { Pin } from "./-links/signup-pin";
import { Done, type Flow, Name, Welcome } from "./-links/signup-steps";
import { type Step, type StepProps, Steps } from "./-links/steps";

export const Route = createFileRoute("/signup")({
  beforeLoad: ({ context, location }) => requireEntry(context, "link", location),
  component: Signup,
});

function Passkey({ next }: StepProps<Flow>) {
  return (
    <PasskeyStep
      onDone={() => {
        next();
      }}
    />
  );
}

const always = () => true;

const steps: readonly Step<Flow>[] = [
  { id: "welcome", shows: always, Step: Welcome },
  { id: "name", shows: always, Step: Name },
  { id: "pin", shows: always, Step: Pin },
  { id: "passkey", shows: (flow) => flow.passkeys.available, Step: Passkey },
  { id: "done", shows: always, Step: Done },
];

function Join({ token, link, linkDied }: LiveProps) {
  const passkeys = usePasskeySupport();
  return <Steps steps={steps} start={{ token, link, passkeys, displayName: "", linkDied }} />;
}

function Signup() {
  return <LinkPage kind="signup" title="Join SlipStream" forSomeoneNew Live={Join} />;
}
