import type { QueryClient } from "@tanstack/react-query";
import { type ParsedLocation, redirect } from "@tanstack/react-router";

import type { IdentityAccount, SystemStatus, SystemStatusClaim } from "../api/gen/model";
import { safeRedirect } from "./safe-redirect";
import { sessionQuery, statusQuery } from "./session";

export type Entry =
  | { readonly kind: "setup" }
  | { readonly kind: "sign-in" }
  | { readonly kind: "worker" }
  | { readonly kind: "app"; readonly account: IdentityAccount }
  | { readonly kind: "portal"; readonly account: IdentityAccount };

export type EntryKind = Entry["kind"];

/** What a route lets in: an entry kind, or `link` for a public link page anyone may open. */
export type Want = EntryKind | "link";

const signedIn = (account: IdentityAccount): Entry =>
  account.admin ? { kind: "app", account } : { kind: "portal", account };

const entries: { readonly [C in SystemStatusClaim]: (account: IdentityAccount | null) => Entry } = {
  unclaimed: () => ({ kind: "setup" }),
  controller: (account) => (account === null ? { kind: "sign-in" } : signedIn(account)),
  worker: () => ({ kind: "worker" }),
};

export const entryOf = (status: SystemStatus, account: IdentityAccount | null): Entry =>
  entries[status.claim](account);

type Cell = (href: string, redirect: string | undefined) => string | null;

const render: Cell = () => null;

const toSignIn: Cell = (href) => `/sign-in?redirect=${encodeURIComponent(href)}`;
const toRedirect: Cell = (_href, to) => safeRedirect(to);

const redirects: { readonly [W in Want]: { readonly [E in EntryKind]: Cell } } = {
  app: {
    setup: () => "/setup",
    "sign-in": toSignIn,
    worker: () => "/worker",
    app: render,
    portal: () => "/portal",
  },
  portal: {
    setup: () => "/setup",
    "sign-in": toSignIn,
    worker: () => "/worker",
    app: render,
    portal: render,
  },
  "sign-in": {
    setup: () => "/setup",
    "sign-in": render,
    worker: () => "/worker",
    app: toRedirect,
    portal: toRedirect,
  },
  setup: {
    setup: render,
    "sign-in": () => "/sign-in",
    worker: () => "/worker",
    app: () => "/",
    portal: () => "/portal",
  },
  worker: {
    setup: () => "/setup",
    "sign-in": () => "/sign-in",
    worker: render,
    app: () => "/",
    portal: () => "/portal",
  },
  link: {
    setup: render,
    "sign-in": render,
    worker: render,
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
    session === null
      ? entryOf(await queryClient.query(statusQuery), null)
      : signedIn(session.account);
  const target = redirectFor(entry, want, location.href, to);
  if (target !== null) {
    redirect({ href: target, replace: true, throw: true });
  }
  return entry;
}
