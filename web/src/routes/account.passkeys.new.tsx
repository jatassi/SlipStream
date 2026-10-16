import { createFileRoute, redirect } from "@tanstack/react-router";

import { requireEntry } from "../session/gate";

// One link for anyone: an Admin's Your Account is in the app and a Member's
// in the portal, and either opens Add a passkey.
export const Route = createFileRoute("/account/passkeys/new")({
  beforeLoad: async ({ context, location }) => {
    const entry = await requireEntry(context, "portal", location);
    redirect({
      to: entry.kind === "app" ? "/account" : "/portal/account",
      search: { add: "passkey" },
      replace: true,
      throw: true,
    });
  },
});
