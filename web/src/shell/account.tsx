import { LibraryIcon, UserGroupIcon } from "@hugeicons/core-free-icons";
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

/** An Admin's way into the portal, to see what Members see, and back. */
export const toPortal = { label: "Open the portal", icon: UserGroupIcon, to: "/portal" } as const;
export const fromPortal = { label: "Back to Library", icon: LibraryIcon, to: "/library" } as const;

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
