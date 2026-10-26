import type { QueryClient } from "@tanstack/react-query";
import { type ParsedLocation, redirect } from "@tanstack/react-router";

import { isProblem } from "../api/errors";
import type {
  FirstrunFirstRun,
  IdentityAccount,
  IdentitySession,
  SystemStatus,
  SystemStatusClaim,
} from "../api/gen/model";
import { type Place, currentPlace, envOf, shownSteps } from "./first-run";
import { safeRedirect } from "./safe-redirect";
import { firstRunQuery, sessionQuery, statusQuery } from "./session";

export type Entry =
  | { readonly kind: "setup" }
  | { readonly kind: "sign-in" }
  | { readonly kind: "worker" }
  | { readonly kind: "unreachable" }
  | { readonly kind: "new-pin"; readonly account: IdentityAccount }
  | { readonly kind: "first-run"; readonly account: IdentityAccount; readonly at: Place }
  | { readonly kind: "app"; readonly account: IdentityAccount }
  | { readonly kind: "portal"; readonly account: IdentityAccount };

export type EntryKind = Entry["kind"];

/** What a route lets in: an entry kind, or `link` for a public link page anyone may open. */
export type Want = EntryKind | "link";

const opensFirstSetup = (session: IdentitySession): boolean =>
  session.account.admin && session.firstRun === "open";

// A passkey sign-in on a turned-off PIN gives a session that can only set a
// new PIN, so it enters nowhere else until it does. An Admin whose first
// setup is open stands on its current place until they finish it.
function signedIn({ account, pinRequired }: IdentitySession, firstSetup: Place | undefined): Entry {
  if (pinRequired) {
    return { kind: "new-pin", account };
  }
  if (firstSetup !== undefined) {
    return { kind: "first-run", account, at: firstSetup };
  }
  return account.admin ? { kind: "app", account } : { kind: "portal", account };
}

type Signed = { readonly session: IdentitySession; readonly firstSetup: Place | undefined };

const entries: { readonly [C in SystemStatusClaim]: (signed: Signed | null) => Entry } = {
  unclaimed: () => ({ kind: "setup" }),
  controller: (signed) =>
    signed === null ? { kind: "sign-in" } : signedIn(signed.session, signed.firstSetup),
  worker: () => ({ kind: "worker" }),
};

const firstSetupAt = (
  status: SystemStatus,
  session: IdentitySession,
  view: FirstrunFirstRun | undefined,
): Place | undefined =>
  view?.open === true && opensFirstSetup(session)
    ? currentPlace(shownSteps(view, envOf(status)))
    : undefined;

export const entryOf = (
  status: SystemStatus,
  session: IdentitySession | null,
  firstRun?: FirstrunFirstRun,
): Entry =>
  entries[status.claim](
    session === null ? null : { session, firstSetup: firstSetupAt(status, session, firstRun) },
  );

type Cell = (href: string, redirect: string | undefined, entry: Entry) => string | null;

const render: Cell = () => null;

const toSignIn: Cell = (href) => `/sign-in?redirect=${encodeURIComponent(href)}`;
const toRedirect: Cell = (_href, to) => safeRedirect(to);
const toNewPin: Cell = () => "/new-pin";
const toServing: Cell = () => "/serving";
const toFirstSetup: Cell = (_href, _to, entry) =>
  entry.kind === "first-run" ? `/setup/${entry.at}` : null;

const redirects: { readonly [W in Want]: { readonly [E in EntryKind]: Cell } } = {
  app: {
    setup: () => "/setup",
    "sign-in": toSignIn,
    worker: () => "/worker",
    "new-pin": toNewPin,
    app: render,
    portal: () => "/portal",
    unreachable: toServing,
    "first-run": toFirstSetup,
  },
  portal: {
    setup: () => "/setup",
    "sign-in": toSignIn,
    worker: () => "/worker",
    "new-pin": toNewPin,
    app: render,
    portal: render,
    unreachable: toServing,
    "first-run": toFirstSetup,
  },
  "sign-in": {
    setup: () => "/setup",
    "sign-in": render,
    worker: () => "/worker",
    "new-pin": toNewPin,
    app: toRedirect,
    portal: toRedirect,
    unreachable: toServing,
    "first-run": toFirstSetup,
  },
  setup: {
    setup: render,
    "sign-in": () => "/sign-in",
    worker: () => "/worker",
    "new-pin": toNewPin,
    app: () => "/",
    portal: () => "/portal",
    unreachable: toServing,
    "first-run": toFirstSetup,
  },
  worker: {
    setup: () => "/setup",
    "sign-in": () => "/sign-in",
    worker: render,
    "new-pin": toNewPin,
    app: () => "/",
    portal: () => "/portal",
    unreachable: toServing,
    "first-run": toFirstSetup,
  },
  "new-pin": {
    setup: () => "/setup",
    "sign-in": () => "/sign-in",
    worker: () => "/worker",
    "new-pin": render,
    app: () => "/",
    portal: () => "/portal",
    unreachable: toServing,
    "first-run": toFirstSetup,
  },
  unreachable: {
    setup: () => "/setup",
    "sign-in": () => "/sign-in",
    worker: () => "/worker",
    "new-pin": toNewPin,
    app: () => "/",
    portal: () => "/portal",
    unreachable: render,
    "first-run": toFirstSetup,
  },
  "first-run": {
    setup: () => "/setup",
    "sign-in": toSignIn,
    worker: () => "/worker",
    "new-pin": toNewPin,
    app: () => "/",
    portal: () => "/portal",
    unreachable: toServing,
    "first-run": render,
  },
  link: {
    setup: render,
    "sign-in": render,
    worker: render,
    "new-pin": render,
    app: render,
    portal: render,
    unreachable: toServing,
    "first-run": render,
  },
};

export const redirectFor = (entry: Entry, want: Want, href: string, to?: string): string | null =>
  redirects[want][entry.kind](href, to, entry);

async function entryNow(queryClient: QueryClient): Promise<Entry> {
  try {
    const session = await queryClient.query({ ...sessionQuery, staleTime: "static" });
    if (session === null) {
      return entryOf(await queryClient.query(statusQuery), null);
    }
    if (!opensFirstSetup(session)) {
      return signedIn(session, undefined);
    }
    const [status, firstRun] = await Promise.all([
      queryClient.query({ ...statusQuery, staleTime: "static" }),
      queryClient.query({ ...firstRunQuery, staleTime: "static" }),
    ]);
    return entryOf(status, session, firstRun);
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
