import { useSuspenseQuery } from "@tanstack/react-query";
import { type ReactNode, useState } from "react";

import type { IdentityLiveLink } from "../../api/gen/model";
import { sessionQuery } from "../../session/session";
import { EntryPage } from "../../shell/entry-page";
import { Spinner } from "../../ui/spinner";
import { DeadLink } from "./dead-link";
import { type LinkKind, kinds, useLink } from "./link";
import { SignedInVisitor } from "./signed-in-visitor";

export type LiveProps = {
  readonly token: string;
  readonly link: IdentityLiveLink;
  /** The Node said the link died while the page was open, so the card shows why. */
  readonly linkDied: () => void;
};

/**
 * A link page: the card for a dead link, or the flow for a live one. `forSomeoneNew`
 * links sign in as their Account, so someone already signed in signs out first.
 */
export function LinkPage({
  kind,
  title,
  forSomeoneNew,
  Live,
}: {
  kind: LinkKind;
  title: string;
  forSomeoneNew: boolean;
  Live: (props: LiveProps) => ReactNode;
}) {
  const { link, recheck } = useLink(kind);
  const session = useSuspenseQuery(sessionQuery).data;
  const [visitor, setVisitor] = useState(session);
  let body: ReactNode;
  if (forSomeoneNew && visitor !== null) {
    body = (
      <SignedInVisitor
        name={visitor.account.displayName}
        onSignedOut={() => {
          setVisitor(null);
        }}
      />
    );
  } else if (link.state === "loading") {
    body = <Spinner label="Opening the link" />;
  } else if (link.state === "dead") {
    body = <DeadLink dead={link.dead} noun={kinds[kind].noun} onTryAgain={recheck} />;
  } else {
    body = <Live key={link.token} token={link.token} link={link.link} linkDied={recheck} />;
  }
  return <EntryPage title={title}>{body}</EntryPage>;
}
