import type { QueryClient } from "@tanstack/react-query";
import { type ParsedLocation, redirect } from "@tanstack/react-router";

import { isProblem } from "../api/errors";
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
  | { readonly kind: "unreachable" }
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
const toServing: Cell = () => "/serving";

const redirects: { readonly [W in Want]: { readonly [E in EntryKind]: Cell } } = {
  app: {
    setup: () => "/setup",
    "sign-in": toSignIn,
    worker: () => "/worker",
    "new-pin": toNewPin,
    app: render,
    portal: () => "/portal",
    unreachable: toServing,
  },
  portal: {
    setup: () => "/setup",
    "sign-in": toSignIn,
    worker: () => "/worker",
    "new-pin": toNewPin,
    app: render,
    portal: render,
    unreachable: toServing,
  },
  "sign-in": {
    setup: () => "/setup",
    "sign-in": render,
    worker: () => "/worker",
    "new-pin": toNewPin,
    app: toRedirect,
    portal: toRedirect,
    unreachable: toServing,
  },
  setup: {
    setup: render,
    "sign-in": () => "/sign-in",
    worker: () => "/worker",
    "new-pin": toNewPin,
    app: () => "/",
    portal: () => "/portal",
    unreachable: toServing,
  },
  worker: {
    setup: () => "/setup",
    "sign-in": () => "/sign-in",
    worker: render,
    "new-pin": toNewPin,
    app: () => "/",
    portal: () => "/portal",
    unreachable: toServing,
  },
  "new-pin": {
    setup: () => "/setup",
    "sign-in": () => "/sign-in",
    worker: () => "/worker",
    "new-pin": render,
    app: () => "/",
    portal: () => "/portal",
    unreachable: toServing,
  },
  unreachable: {
    setup: () => "/setup",
    "sign-in": () => "/sign-in",
    worker: () => "/worker",
    "new-pin": toNewPin,
    app: () => "/",
    portal: () => "/portal",
    unreachable: render,
  },
  link: {
    setup: render,
    "sign-in": render,
    worker: render,
    "new-pin": render,
    app: render,
    portal: render,
    unreachable: toServing,
  },
};

export const redirectFor = (entry: Entry, want: Want, href: string, to?: string): string | null =>
  redirects[want][entry.kind](href, to);

async function entryNow(queryClient: QueryClient): Promise<Entry> {
  try {
    const session = await queryClient.query({ ...sessionQuery, staleTime: "static" });
    return session === null
      ? entryOf(await queryClient.query(statusQuery), null)
      : signedIn(session);
  } catch (error) {
    if (isProblem(error, "controller_unreachable")) {
      return { kind: "unreachable" };
    }
    throw error;
  }
}

export async function requireEntry(
  { queryClient }: { queryClient: QueryClient },
  want: Want,
  location: ParsedLocation,
  to?: string,
): Promise<Entry> {
  const entry = await entryNow(queryClient);
  const target = redirectFor(entry, want, location.href, to);
  if (target !== null) {
    redirect({ href: target, replace: true, throw: true });
  }
  return entry;
}
