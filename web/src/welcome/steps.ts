import type { HistoryState } from "@tanstack/react-router";
import type { ComponentType } from "react";

import { WatchStep } from "../media-servers/access/watch-step";
import { DoneStep, PasskeyStep } from "./screens";

declare module "@tanstack/react-router" {
  interface HistoryState {
    /** Set by the signup page, which counts its own steps before these. */
    readonly signup?: { readonly shown: number };
  }
}

/**
 * What decides which welcome steps show. The signup page reads them from its
 * link before the Account exists, and `/welcome` from the Member's access, so
 * both count the same steps.
 */
export type WelcomeFacts = {
  readonly passkeys: boolean;
  /** How many Media Servers the Member is offered. */
  readonly servers: number;
};

export type WelcomeStepProps = { readonly next: () => void };

/** One row of the steps after signup. Later specs add rows, so this is a table and not branches. */
export type WelcomeRow = {
  readonly id: string;
  readonly title: string;
  readonly shows: (facts: WelcomeFacts) => boolean;
  readonly Step: ComponentType<WelcomeStepProps>;
};

export const welcomeSteps: readonly WelcomeRow[] = [
  { id: "passkey", title: "Add a passkey", shows: (facts) => facts.passkeys, Step: PasskeyStep },
  { id: "watch", title: "Watch", shows: (facts) => facts.servers > 0, Step: WatchStep },
  // S42 (Portal inbox and Member notifications) inserts "email" here, S41 (Member portal) "request"
  { id: "done", title: "Welcome", shows: () => true, Step: DoneStep },
];

export const shownWelcome = (rows: readonly WelcomeRow[], facts: WelcomeFacts) =>
  rows.filter((row) => row.shows(facts));

/** The step a URL names when it shows, else the first step. */
export function placeWelcome(shown: readonly WelcomeRow[], wanted?: string): WelcomeRow {
  const at = shown.find((row) => row.id === wanted) ?? shown[0];
  if (at === undefined) {
    throw new Error("the welcome steps show no step");
  }
  return at;
}

export const nextWelcome = (shown: readonly WelcomeRow[], at: WelcomeRow): WelcomeRow =>
  shown[shown.indexOf(at) + 1] ?? at;

/** How many signup steps came before, when this tab came from `/signup`, for "Step N of M". */
export const countFrom = (state: HistoryState): number | undefined => state.signup?.shown;
