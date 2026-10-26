import { useQuery, useSuspenseQuery } from "@tanstack/react-query";
import { useState } from "react";

import { statusQuery } from "../../session/session";
import type { SetupCode } from "../../session/setup-code";
import { EntryPage } from "../../shell/entry-page";
import { ChoiceList, choices } from "./choices";
import { CheckingCode } from "./code-screen";
import { Removed } from "./removed";
import { type Screen, opening, useJoinOutcome, workerClaimQuery } from "./worker-claim";

export const setupTitle = "Set up SlipStream";

/** The screens after the Setup code: the choice, then the chosen form. */
export function Chosen({
  code,
  linked,
  restarted,
}: {
  code: SetupCode;
  linked: boolean;
  restarted: () => void;
}) {
  const { data: status } = useSuspenseQuery(statusQuery);
  const claim = useQuery(workerClaimQuery(code));
  const [chosen, setChosen] = useState<Screen>();
  useJoinOutcome(claim, restarted);
  const screen = chosen ?? opening(claim);
  if (screen === undefined) {
    return (
      <EntryPage title={setupTitle}>
        <CheckingCode />
      </EntryPage>
    );
  }
  if (screen === "choice") {
    return (
      <EntryPage title={setupTitle}>
        <Removed />
        {linked ? <p className="text-meta text-ink-3">Setup code filled in</p> : null}
        <ChoiceList status={status} choose={setChosen} />
      </EntryPage>
    );
  }
  const { label, Form } = choices[screen];
  return (
    <EntryPage title={label}>
      <Form
        code={code}
        back={() => {
          setChosen("choice");
        }}
        open={setChosen}
        restarted={restarted}
      />
    </EntryPage>
  );
}
