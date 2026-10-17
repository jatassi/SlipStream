import type { QueryClient } from "@tanstack/react-query";
import { type ParsedLocation, redirect } from "@tanstack/react-router";

import type {
  IdentityAccount,
  IdentitySession,
  SystemStatus,
  SystemStatusClaim,
} from "../api/gen/model";
import { safeRedirect } from "./safe-redirect";
import { sessionQuery, statusQuery } from "./session";

export type Entry =
  | { readonly kind: "setup" }
  | { readonly kind: "sign-in" }
  | { readonly kind: "worker" }
  | { readonly kind: "new-pin"; readonly account: IdentityAccount }
  | { readonly kind: "app"; readonly account: IdentityAccount }
  | { readonly kind: "portal"; readonly account: IdentityAccount };

export type EntryKind = Entry["kind"];

/** What a route lets in: an entry kind, or `link` for a public link page anyone may open. */
export type Want = EntryKind | "link";

// A passkey sign-in on a turned-off PIN gives a session that can only set a
// new PIN, so it enters nowhere else until it does.
function signedIn({ account, pinRequired }: IdentitySession): Entry {
  if (pinRequired) {
    return { kind: "new-pin", account };
  }
  return account.admin ? { kind: "app", account } : { kind: "portal", account };
}

const entries: { readonly [C in SystemStatusClaim]: (session: IdentitySession | null) => Entry } = {
  unclaimed: () => ({ kind: "setup" }),
  controller: (session) => (session === null ? { kind: "sign-in" } : signedIn(session)),
  worker: () => ({ kind: "worker" }),
};

export const entryOf = (status: SystemStatus, session: IdentitySession | null): Entry =>
  entries[status.claim](session);

type Cell = (href: string, redirect: string | undefined) => string | null;

const render: Cell = () => null;

const toSignIn: Cell = (href) => `/sign-in?redirect=${encodeURIComponent(href)}`;
const toRedirect: Cell = (_href, to) => safeRedirect(to);
const toNewPin: Cell = () => "/new-pin";

const redirects: { readonly [W in Want]: { readonly [E in EntryKind]: Cell } } = {
  app: {
    setup: () => "/setup",
    "sign-in": toSignIn,
    worker: () => "/worker",
    "new-pin": toNewPin,
    app: render,
    portal: () => "/portal",
  },
  portal: {
    setup: () => "/setup",
    "sign-in": toSignIn,
    worker: () => "/worker",
    "new-pin": toNewPin,
    app: render,
    portal: render,
  },
  "sign-in": {
    setup: () => "/setup",
    "sign-in": render,
    worker: () => "/worker",
    "new-pin": toNewPin,
    app: toRedirect,
    portal: toRedirect,
  },
  setup: {
    setup: render,
    "sign-in": () => "/sign-in",
    worker: () => "/worker",
    "new-pin": toNewPin,
    app: () => "/",
    portal: () => "/portal",
  },
  worker: {
    setup: () => "/setup",
    "sign-in": () => "/sign-in",
    worker: render,
    "new-pin": toNewPin,
    app: () => "/",
    portal: () => "/portal",
  },
  "new-pin": {
    setup: () => "/setup",
    "sign-in": () => "/sign-in",
    worker: () => "/worker",
    "new-pin": render,
    app: () => "/",
    portal: () => "/portal",
  },
  link: {
    setup: render,
    "sign-in": render,
    worker: render,
    "new-pin": render,
    app: render,
    portal: render,
  },
};

export const redirectFor = (entry: Entry, want: Want, href: string, to?: string): string | null =>
  redirects[want][entry.kind](href, to);

export async function requireEntry(
  { queryClient }: { queryClient: QueryClient },
  want: Want,
  location: ParsedLocation,
  to?: string,
): Promise<Entry> {
  const session = await queryClient.query({ ...sessionQuery, staleTime: "static" });
  const entry: Entry =
    session === null ? entryOf(await queryClient.query(statusQuery), null) : signedIn(session);
  const target = redirectFor(entry, want, location.href, to);
  if (target !== null) {
    redirect({ href: target, replace: true, throw: true });
  }
  return entry;
}
