import { createFileRoute } from "@tanstack/react-router";

import type { IdentitySignedIn } from "../api/gen/model";
import type { SessionEndReason } from "../live/types";
import { requireEntry } from "../session/gate";
import { signInSearch } from "../session/safe-redirect";
import { EntryPage } from "../shell/entry-page";
import { SignInForm } from "./-sign-in/sign-in-form";
import { signedInTo } from "./-sign-in/signed-in";

const reasonText: Record<SessionEndReason, string> = {
  "signed-out": "You signed out.",
  expired: "Your session ended. Sign in again.",
  revoked: "This device was signed out.",
};

export const Route = createFileRoute("/sign-in")({
  validateSearch: signInSearch,
  beforeLoad: ({ context, location, search: { redirect } }) =>
    requireEntry(context, "sign-in", location, redirect),
  component: SignIn,
});

function SignIn() {
  const { reason, redirect } = Route.useSearch();
  const { queryClient } = Route.useRouteContext();
  const navigate = Route.useNavigate();
  const signedIn = async (session: IdentitySignedIn) => {
    await navigate({ href: signedInTo(queryClient, session, redirect) });
  };
  return (
    <EntryPage title="Sign in to SlipStream">
      {reason === undefined ? null : <output className="text-ink-1">{reasonText[reason]}</output>}
      <SignInForm signedIn={signedIn} />
      <p className="text-meta text-ink-3">Forgot your PIN? Ask an Admin for a Reset link.</p>
    </EntryPage>
  );
}
