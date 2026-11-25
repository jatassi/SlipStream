import { createFileRoute } from "@tanstack/react-router";

import { requireEntry } from "../session/gate";
import { usePasskeySupport } from "../session/passkey";
import { placeWelcome, shownWelcome, welcomeSteps } from "../welcome/steps";
import { type LiveProps, LinkPage } from "./-links/link-page";
import { Pin } from "./-links/signup-pin";
import { type Flow, Name, Welcome } from "./-links/signup-steps";
import { type Step, Steps } from "./-links/steps";

export const Route = createFileRoute("/signup")({
  beforeLoad: ({ context, location }) => requireEntry(context, "link", location),
  component: Signup,
});

const always = () => true;

const steps: readonly Step<Flow>[] = [
  { id: "welcome", shows: always, Step: Welcome },
  { id: "name", shows: always, Step: Name },
  { id: "pin", shows: always, Step: Pin },
];

function Join({ token, link, linkDied }: LiveProps) {
  const passkeys = usePasskeySupport();
  const after = shownWelcome(welcomeSteps, {
    passkeys: passkeys.available,
    servers: link.mediaServers?.length ?? 0,
  });
  const welcome = { step: placeWelcome(after).id, before: steps.length };
  return (
    <Steps
      steps={steps}
      after={after.length}
      start={{ token, link, displayName: "", linkDied, welcome }}
    />
  );
}

function Signup() {
  return <LinkPage kind="signup" title="Join SlipStream" forSomeoneNew Live={Join} />;
}
