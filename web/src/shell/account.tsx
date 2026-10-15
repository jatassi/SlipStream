import { useRouteContext } from "@tanstack/react-router";

import { useDeleteSession } from "../api/gen/query/identity/identity";
import { usePref, writePref } from "./prefs";

export function useSignOut(): () => void {
  const endSession = useRouteContext({ from: "__root__", select: (context) => context.endSession });
  const signOut = useDeleteSession({
    mutation: {
      onSuccess: () => {
        endSession("signed-out");
      },
    },
  });
  return () => {
    signOut.mutate();
  };
}

export const startupLabel = "Startup sequence";

export function useStartupOn(): [on: boolean, set: (on: boolean) => void] {
  return [
    !usePref("startupOff"),
    (on) => {
      writePref("startupOff", !on);
    },
  ];
}

export function Avatar({ name }: { name: string }) {
  return (
    <span
      aria-hidden="true"
      className="grid size-7 shrink-0 place-items-center rounded-capsule bg-fill-hover text-chip text-ink-1 ring-1 ring-hairline-strong ring-inset"
    >
      {name.trim().charAt(0).toUpperCase()}
    </span>
  );
}
