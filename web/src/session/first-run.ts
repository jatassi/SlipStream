import type {
  FirstrunFirstRun,
  FirstrunStepID,
  FirstrunStepState,
  SystemStatus,
} from "../api/gen/model";
import { passkeySupport } from "./passkey";

/** Where an Admin stands in first setup: a step, or the done screen once no step is todo. */
export type Place = FirstrunStepID | "done";

/** What this browser at this address can do, for the steps only the web can hide. */
export type FirstRunEnv = { readonly passkeys: boolean };

export const envOf = (status: SystemStatus): FirstRunEnv => ({
  passkeys: passkeySupport(status.passkeys).available,
});

// The server lists every step whose facts allow it. S12's passkey step also
// needs a browser and an address that can hold a passkey, which only the
// browser knows.
const hiddenOnWeb: { readonly [S in FirstrunStepID]?: (env: FirstRunEnv) => boolean } = {
  passkey: (env) => !env.passkeys,
};

export const shownSteps = (
  view: FirstrunFirstRun,
  env: FirstRunEnv,
): readonly FirstrunStepState[] =>
  view.steps.filter((step) => hiddenOnWeb[step.id]?.(env) !== true);

export const currentPlace = (shown: readonly FirstrunStepState[]): Place =>
  shown.find((step) => step.state === "todo")?.id ?? "done";

/** The place a URL names when the Admin may stand there, else where first setup holds them. */
export function placeAt(shown: readonly FirstrunStepState[], wanted: string): Place {
  const current = currentPlace(shown);
  if (wanted === "done") {
    return current;
  }
  return shown.find((step) => step.id === wanted)?.id ?? current;
}

export type Around = {
  /** 1-based, for "Step 2 of 5"; the done screen has none. */
  readonly number: number | undefined;
  readonly count: number;
  readonly back: Place | undefined;
  readonly next: Place;
  readonly ahead: readonly FirstrunStepID[];
};

export function around(shown: readonly FirstrunStepState[], at: Place): Around {
  const index = at === "done" ? shown.length : shown.findIndex((step) => step.id === at);
  if (index === -1) {
    throw new Error(`first setup shows no ${at} step`);
  }
  return {
    number: at === "done" ? undefined : index + 1,
    count: shown.length,
    back: shown[index - 1]?.id,
    next: shown[index + 1]?.id ?? "done",
    ahead: shown.slice(index + 1).map((step) => step.id),
  };
}
