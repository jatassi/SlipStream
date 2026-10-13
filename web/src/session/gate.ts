import type { QueryClient } from "@tanstack/react-query";
import { type ParsedLocation, redirect } from "@tanstack/react-router";

import type { IdentityAccount, SystemStatus, SystemStatusClaim } from "../api/gen/model";
import { safeRedirect } from "./safe-redirect";
import { sessionQuery, statusQuery } from "./session";

export type Entry =
  | { readonly kind: "setup" }
  | { readonly kind: "sign-in" }
  | { readonly kind: "worker" }
  | { readonly kind: "app"; readonly account: IdentityAccount };

export type EntryKind = Entry["kind"];

const entries: { readonly [C in SystemStatusClaim]: (account: IdentityAccount | null) => Entry } = {
  unclaimed: () => ({ kind: "setup" }),
  controller: (account) => (account === null ? { kind: "sign-in" } : { kind: "app", account }),
  worker: () => ({ kind: "worker" }),
};

export const entryOf = (status: SystemStatus, account: IdentityAccount | null): Entry =>
  entries[status.claim](account);

type Cell = (href: string, redirect: string | undefined) => string | null;

const render: Cell = () => null;

const redirects: { readonly [W in EntryKind]: { readonly [E in EntryKind]: Cell } } = {
  app: {
    setup: () => "/setup",
    "sign-in": (href) => `/sign-in?redirect=${encodeURIComponent(href)}`,
    worker: () => "/worker",
    app: render,
  },
  "sign-in": {
    setup: () => "/setup",
    "sign-in": render,
    worker: () => "/worker",
    app: (_href, to) => safeRedirect(to),
  },
  setup: {
    setup: render,
    "sign-in": () => "/sign-in",
    worker: () => "/worker",
    app: () => "/",
  },
  worker: {
    setup: () => "/setup",
    "sign-in": () => "/sign-in",
    worker: render,
    app: () => "/",
  },
};

export const redirectFor = (
  entry: Entry,
  want: EntryKind,
  href: string,
  to?: string,
): string | null => redirects[want][entry.kind](href, to);

export async function requireEntry(
  { queryClient }: { queryClient: QueryClient },
  want: EntryKind,
  location: ParsedLocation,
  to?: string,
): Promise<Entry> {
  const session = await queryClient.query({ ...sessionQuery, staleTime: "static" });
  const entry: Entry =
    session === null
      ? entryOf(await queryClient.query(statusQuery), null)
      : { kind: "app", account: session.account };
  const target = redirectFor(entry, want, location.href, to);
  if (target !== null) {
    redirect({ href: target, replace: true, throw: true });
  }
  return entry;
}
