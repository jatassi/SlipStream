import { Alert02Icon, LockedIcon } from "@hugeicons/core-free-icons";
import { type ReactNode, useState } from "react";

import { UnreachableError, isApiError } from "../api/errors";
import { Data } from "./data";
import { Icon } from "./icon";
import type { Setting } from "./setting-kind";

export function Refusal({ error }: { error: string }) {
  return (
    <p
      role="alert"
      className="flex items-center gap-1.5 text-meta text-fail-ink in-data-[density=touch]:text-body-touch"
    >
      <Icon icon={Alert02Icon} className="size-3.5 shrink-0" />
      {error}
    </p>
  );
}

export function Lock({ variable }: { variable: string }) {
  return (
    <p className="flex items-center gap-1.5 text-meta text-ink-2 in-data-[density=touch]:text-body-touch">
      <Icon icon={LockedIcon} className="size-3.5 shrink-0" />
      <span>
        Set by <Data>{variable}</Data>
      </span>
    </p>
  );
}

export const appliesHint: { readonly [A in Setting["applies"]]: readonly string[] } = {
  live: [],
  restart: ["Applies after restart."],
};

export function Hints({ hints, reset }: { hints: readonly string[]; reset: ReactNode }) {
  return (
    <div className="flex flex-wrap items-center justify-between gap-x-3 gap-y-1">
      <div className="flex min-w-0 flex-col gap-0.5 text-meta text-ink-3 in-data-[density=touch]:text-body-touch">
        {hints.map((hint) => (
          <p key={hint}>{hint}</p>
        ))}
      </div>
      {reset}
    </div>
  );
}

function refusal(error: unknown): string {
  if (isApiError(error)) {
    return error.detail ?? error.title;
  }
  if (error instanceof UnreachableError) {
    return error.message;
  }
  throw error;
}

export function useSave<V>(onSave: (value: V | null) => Promise<unknown>, onSaved: () => void) {
  const [error, setError] = useState<string>();
  const [busy, setBusy] = useState(false);
  const send = async (value: V | null) => {
    setBusy(true);
    setError(undefined);
    try {
      await onSave(value);
      onSaved();
    } catch (failure: unknown) {
      setError(refusal(failure));
    } finally {
      setBusy(false);
    }
  };
  return { error, setError, busy, send };
}
