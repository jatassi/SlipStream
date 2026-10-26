import { useQuery } from "@tanstack/react-query";
import { useState } from "react";

import type { PairingWorkerClaim } from "../../api/gen/model";
import { Spinner } from "../../ui/spinner";
import type { FormProps } from "./form-props";
import { JoinCodeForm } from "./join-code-form";
import { type JoinRefusal, joinSentences } from "./join-sentences";
import { JoinFailed, JoinWait } from "./join-wait";
import { AlreadySetUp, Refusal } from "./refusal";
import { workerClaimQuery } from "./worker-claim";

/** Join an existing SlipStream with a Join Code, waiting here for the dial when needed. */
export function JoinForm(props: FormProps) {
  const claim = useQuery(workerClaimQuery(props.code));
  const [dismissed, setDismissed] = useState<PairingWorkerClaim>();
  const [refused, setRefused] = useState<JoinRefusal>();
  const { data } = claim;
  if (refused !== undefined) {
    return joinSentences[refused].next === "sign-in" ? (
      <AlreadySetUp />
    ) : (
      <Refusal
        text={joinSentences[refused].text}
        action="Back to the wait"
        onAction={() => {
          setRefused(undefined);
          void claim.refetch();
        }}
      />
    );
  }
  if (data === undefined && claim.isFetching) {
    return <Spinner label="Reading this Node's Join" />;
  }
  if (data?.state === "awaiting-dial") {
    return <JoinWait claim={data} />;
  }
  if (data?.state === "failed" && data !== dismissed) {
    return (
      <JoinFailed
        problem={data.problem}
        onAnother={() => {
          setDismissed(data);
        }}
      />
    );
  }
  return <JoinCodeForm props={props} outcomes={{ waiting: claim.refetch, refused: setRefused }} />;
}
