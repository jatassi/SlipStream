import { Link } from "@tanstack/react-router";

import { AlertLine } from "../../ui/alert-line";
import { Button } from "../../ui/button";
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
      <AlertLine text={text} />
      <Button onClick={onAction}>{action}</Button>
    </div>
  );
}

export function AlreadySetUp() {
  return (
    <div className="flex flex-col items-start gap-3">
      <AlertLine text={alreadySetUpSentence} />
      <Button nativeButton={false} render={<Link to="/sign-in" />}>
        Sign in
      </Button>
    </div>
  );
}
