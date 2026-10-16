import { Add01Icon, Link01Icon, Logout01Icon, SidebarLeftIcon } from "@hugeicons/core-free-icons";
import type { IconSvgElement } from "@hugeicons/react";
import { type UseNavigateResult, useNavigate } from "@tanstack/react-router";

import { type ShellKind, useShell } from "../ui/shell-context";
import { addLabel } from "../works/fields";
import { type NavRow, useAdminNav, useContentTypes } from "./nav";
import { askSignOut } from "./sign-out";
import { toggleSidebar } from "./sidebar-state";

export type CommandContext = {
  readonly navigate: UseNavigateResult<string>;
};

export type Command = {
  readonly id: string;
  readonly label: string;
  readonly meta?: string;
  readonly icon: IconSvgElement;
  readonly audience: NavRow["audience"];
  readonly shell?: ShellKind;
  readonly run: (context: CommandContext) => void;
};

const commands: readonly Command[] = [
  {
    id: "toggle-sidebar",
    label: "Toggle sidebar",
    icon: SidebarLeftIcon,
    audience: "admin",
    shell: "desktop",
    run: toggleSidebar,
  },
  {
    id: "new-signup-link",
    label: "New Signup link",
    icon: Link01Icon,
    audience: "admin",
    run: ({ navigate }) => {
      void navigate({ to: "/accounts", search: { newLink: true } });
    },
  },
  {
    id: "sign-out",
    label: "Sign out",
    icon: Logout01Icon,
    audience: "admin",
    run: askSignOut,
  },
];

function useGoToCommands(): Command[] {
  return useAdminNav().flatMap((row) => {
    const goToRow: Command = {
      id: `go-${row.id}`,
      label: `Go to ${row.label}`,
      icon: row.icon,
      audience: row.audience,
      run: ({ navigate }) => {
        void navigate({ to: row.to });
      },
    };
    return [goToRow].concat(
      row.children.map((child) => ({
        id: `go-${row.id}-${child.id}`,
        label: `Go to ${child.label}`,
        meta: row.label,
        icon: child.icon,
        audience: row.audience,
        run: ({ navigate }: CommandContext) => {
          void navigate({ to: child.to, params: child.params });
        },
      })),
    );
  });
}

function useAddCommands(): Command[] {
  return useContentTypes()
    .filter((type) => type.enabled)
    .map((type) => ({
      id: `add-${type.id}`,
      label: addLabel(type),
      icon: Add01Icon,
      audience: "admin",
      run: ({ navigate }) => {
        void navigate({ to: "/add", search: { contentType: type.id } });
      },
    }));
}

export function useCommandContext(): CommandContext {
  return { navigate: useNavigate() };
}

export function useAdminCommands(): Command[] {
  const shell = useShell();
  return [...useGoToCommands(), ...useAddCommands(), ...commands].filter(
    (command) =>
      command.audience === "admin" && (command.shell === undefined || command.shell === shell),
  );
}
