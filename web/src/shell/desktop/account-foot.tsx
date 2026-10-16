import { Logout01Icon } from "@hugeicons/core-free-icons";
import { useNavigate } from "@tanstack/react-router";

import { useAccount } from "../../session/session";
import { Menu } from "../../ui/menu";
import { Avatar, startupLabel, toPortal, useStartupOn } from "../account";
import { askSignOut } from "../sign-out";

export function AccountFoot({ collapsed }: { collapsed: boolean }) {
  const account = useAccount();
  const navigate = useNavigate();
  const [startupOn, setStartupOn] = useStartupOn();
  return (
    <div className="shrink-0 px-2.5 pt-2 pb-3 shadow-[inset_0_1px_0_var(--color-hairline)]">
      <Menu
        label="Account"
        trigger={
          <button
            type="button"
            aria-label="Account"
            className={`flex w-full items-center gap-2.5 rounded-item p-1.5 text-left hover:bg-fill ${collapsed ? "justify-center" : ""}`}
          >
            <Avatar name={account.displayName} />
            {!collapsed && <AccountName name={account.displayName} />}
          </button>
        }
        items={[
          {
            label: toPortal.label,
            icon: toPortal.icon,
            onSelect: () => {
              void navigate({ to: toPortal.to });
            },
          },
          {
            kind: "toggle",
            label: startupLabel,
            checked: startupOn,
            onCheckedChange: setStartupOn,
          },
          "separator",
          { label: "Sign out", icon: Logout01Icon, onSelect: askSignOut },
        ]}
      />
    </div>
  );
}

function AccountName({ name }: { name: string }) {
  return (
    <span className="min-w-0 flex-1">
      <span className="block truncate text-body font-medium text-ink-1">{name}</span>
      <span className="block text-meta text-ink-3">Admin</span>
    </span>
  );
}
