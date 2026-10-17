import { createFileRoute } from "@tanstack/react-router";

import type { IdentityPasskey } from "../api/gen/model";
import { requireEntry } from "../session/gate";
import { usePasskeySupport } from "../session/passkey";
import { type LiveProps, LinkPage } from "./-links/link-page";
import { OpenSlipStream } from "./-links/open-slipstream";
import { PasskeyStep } from "./-links/passkey-step";
import { Passkeys } from "./-links/reset-passkeys";
import { type Flow, NewPin } from "./-links/reset-pin";
import { type Step, type StepProps, Steps } from "./-links/steps";

export const Route = createFileRoute("/reset")({
  beforeLoad: ({ context, location }) => requireEntry(context, "link", location),
  component: Reset,
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

function Done() {
  return <OpenSlipStream sentence="Your new PIN is set, and you're signed in." to="/" />;
}

const always = () => true;
const holdsHere = (passkeys: readonly IdentityPasskey[]) =>
  passkeys.some((passkey) => passkey.host === location.hostname);

const steps: readonly Step<Flow>[] = [
  { id: "pin", shows: always, Step: NewPin },
  { id: "passkeys", shows: (flow) => flow.passkeys.length > 0, Step: Passkeys },
  {
    id: "passkey",
    shows: (flow) => flow.support.available && !holdsHere(flow.passkeys),
    Step: Passkey,
  },
  { id: "done", shows: always, Step: Done },
];

function ResetPin({ token, link, linkDied }: LiveProps) {
  const support = usePasskeySupport();
  return (
    <Steps steps={steps} start={{ token, email: link.email, support, passkeys: [], linkDied }} />
  );
}

function Reset() {
  return <LinkPage kind="reset" title="Set a new PIN" forSomeoneNew Live={ResetPin} />;
}
