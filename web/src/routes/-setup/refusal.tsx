import { Link } from "@tanstack/react-router";

import { Button } from "../../ui/button";
import { SignalLine } from "../../ui/signal-line";
import { alreadySetUpSentence } from "./join-sentences";

/** A refusal's sentence with the one way on that it offers. */
export function Refusal({
  text,
  action,
  onAction,
}: {
  text: string;
  action: string;
  onAction: () => void;
}) {
  return (
    <div className="flex flex-col items-start gap-3">
      <SignalLine tone="fail" announce>
        {text}
      </SignalLine>
      <Button onClick={onAction}>{action}</Button>
    </div>
  );
}

export function AlreadySetUp() {
  return (
    <div className="flex flex-col items-start gap-3">
      <SignalLine tone="fail" announce>
        {alreadySetUpSentence}
      </SignalLine>
      <Button nativeButton={false} render={<Link to="/sign-in" />}>
        Sign in
      </Button>
    </div>
  );
}
