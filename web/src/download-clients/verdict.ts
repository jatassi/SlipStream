import type { DownloadclientsClientCard as Card } from "../api/gen/model";
import { portOf } from "../routes/_app/nodes/-node";
import type { Signal } from "../ui/signal-chip";

/** Where a Node's built-in client stands with the peers that would connect in. */
export type Verdict =
  | { readonly kind: "offline" }
  | { readonly kind: "unreported" }
  | { readonly kind: "not-listening"; readonly listen: string; readonly reason?: string }
  | { readonly kind: "reachable"; readonly bound: string }
  | { readonly kind: "unproven"; readonly bound: string };

export function verdictOf(card: Card): Verdict {
  const { client } = card;
  if (!card.inStep) {
    return { kind: "offline" };
  }
  if (client === undefined) {
    return { kind: "unreported" };
  }
  if (client.bound === undefined) {
    return {
      kind: "not-listening",
      listen: client.listen,
      ...(client.listenError !== undefined && { reason: client.listenError }),
    };
  }
  return client.reachedAt === undefined
    ? { kind: "unproven", bound: client.bound }
    : { kind: "reachable", bound: client.bound };
}

export type VerdictWords = {
  readonly signal?: Signal;
  readonly chip?: string;
  readonly sentence: string;
  readonly hint?: string;
};

export function verdictWords(verdict: Verdict, node: string): VerdictWords {
  switch (verdict.kind) {
    case "offline":
      return {
        sentence: `${node} is offline, so its client can't say how it is doing.`,
        hint: "Its settings reach it when it is back.",
      };
    case "unreported":
      return { sentence: `${node}'s client hasn't reported yet.` };
    case "not-listening":
      return {
        signal: "fail",
        chip: "Not listening",
        sentence:
          verdict.reason === undefined
            ? `Not listening on ${verdict.listen} yet.`
            : `Can't listen on ${verdict.listen}: ${verdict.reason}.`,
        hint: "No peer can connect in until it listens. If another program holds the port, choose another one or free it, then restart.",
      };
    case "reachable":
      return {
        signal: "ok",
        chip: "Reachable",
        sentence: `Listening on ${verdict.bound}. Peers from the internet have reached port ${portOf(verdict.bound)}.`,
      };
    case "unproven":
      return {
        sentence: `Listening on ${verdict.bound}. No peer from the internet has connected in yet, so port ${portOf(verdict.bound)} is unproven.`,
        hint: `If it stays unproven, check that your router forwards port ${portOf(verdict.bound)} to this computer. Downloads still work, from fewer peers.`,
      };
    default:
      return verdict satisfies never;
  }
}

/** What the disk guard says while it holds a Node's new downloads, in the words Health uses. */
export function guardWords(card: Card): VerdictWords | undefined {
  const guard = card.diskGuard;
  return (
    guard && {
      signal: "warn",
      chip: "Waiting for disk",
      sentence: guard.sentence,
      hint: guard.hint,
    }
  );
}

/** True for an address such as 0.0.0.0:6881 or [::]:6881, which binds where the computer reaches the internet from. */
export const unspecified = (listen: string): boolean => /^(?:0\.0\.0\.0|\[::\]):\d+$/u.test(listen);
