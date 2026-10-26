import { createFileRoute } from "@tanstack/react-router";

import { requireEntry } from "../session/gate";
import { EntryPage } from "../shell/entry-page";
import { Chosen, setupTitle } from "./-setup/chosen";
import { CheckingCode, CodeScreen } from "./-setup/code-screen";
import { Removed } from "./-setup/removed";
import { useSetupCode } from "./-setup/use-setup-code";

export const Route = createFileRoute("/setup")({
  beforeLoad: ({ context, location }) => requireEntry(context, "setup", location),
  component: Setup,
});

function Setup() {
  const { state, accept, restarted } = useSetupCode();
  if (state.at === "accepted") {
    return <Chosen code={state.code} linked={state.linked} restarted={restarted} />;
  }
  return (
    <EntryPage title={setupTitle}>
      <Removed />
      {state.at === "checking" ? (
        <CheckingCode />
      ) : (
        <CodeScreen note={state.note} accept={accept} />
      )}
    </EntryPage>
  );
}
