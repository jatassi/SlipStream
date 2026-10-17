import { LibraryIcon, Logout01Icon, UserGroupIcon } from "@hugeicons/core-free-icons";
import type { IconSvgElement } from "@hugeicons/react";

import { usePref, writePref } from "./prefs";
import { restartSelf, useRestartSelf } from "./restart/use-restart-self";
import { askSignOut } from "./sign-out";

export const startupLabel = "Startup sequence";

/** An Admin's way into the portal, to see what Members see, and back. */
export const toPortal = { label: "Open the portal", icon: UserGroupIcon, to: "/portal" } as const;
export const fromPortal = { label: "Back to Library", icon: LibraryIcon, to: "/library" } as const;

/** The account menu's closing items, the same on desktop and phone. */
export function useLeaveActions(): readonly {
  label: string;
  icon: IconSvgElement;
  onSelect: () => void;
}[] {
  const restart = useRestartSelf();
  return [
    { label: restartSelf.label, icon: restartSelf.icon, onSelect: restart },
    { label: "Sign out", icon: Logout01Icon, onSelect: askSignOut },
  ];
}

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
