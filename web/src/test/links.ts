import { HttpResponse, http } from "msw";

import type { IdentityLinkView, IdentitySignedIn } from "../api/gen/model";
import { grace, problem } from "./handlers";

export const liveLink = (email = grace.email): IdentityLinkView => ({
  state: "live",
  adminName: "Ada Okafor",
  email,
  serverName: "forge",
  expiresAt: "2026-10-23T10:00:00Z",
});

export const graceSignedIn: IdentitySignedIn = {
  account: grace,
  pinRequired: false,
  keyPrompt: "ask",
  passkeyPrompt: false,
};

/** A signed-out browser holding a link, whose redeem signs in as Grace. */
export function linkServer(
  kind: "signup-links" | "reset-links" | "email-links",
  views: IdentityLinkView[],
) {
  let signedIn = false;
  let gets = 0;
  return {
    gets: () => gets,
    signIn: () => {
      signedIn = true;
    },
    handlers: [
      http.get("*/api/v1/session", () =>
        signedIn
          ? HttpResponse.json(graceSignedIn)
          : problem(401, "signed_out", "Sign in to continue"),
      ),
      http.get(`*/api/v1/${kind}/by-token/:token`, ({ params }) => {
        const view = views[Math.min(gets, views.length - 1)];
        gets += 1;
        return params["token"] === "tok-1" && view !== undefined
          ? HttpResponse.json(view)
          : HttpResponse.json({ state: "dead", reason: "unknown" });
      }),
    ],
  };
}
