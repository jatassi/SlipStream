import { Logout01Icon, UserCircleIcon } from "@hugeicons/core-free-icons";
import { useNavigate } from "@tanstack/react-router";

import { useAccount } from "../../session/session";
import { Menu } from "../../ui/menu";
import { Avatar, fromPortal } from "../account";
import { askSignOut } from "../sign-out";

export function AccountMenu() {
  const { displayName, admin } = useAccount();
  const navigate = useNavigate();
  return (
    <Menu
      label="Account"
      trigger={
        <button
          type="button"
          aria-label="Account"
          className="grid size-11 place-items-center rounded-capsule hover:bg-fill"
        >
          <Avatar name={displayName} />
        </button>
      }
      items={[
        {
          label: "Account",
          icon: UserCircleIcon,
          onSelect: () => {
            void navigate({ to: "/portal/account" });
          },
        },
        ...(admin
          ? [
              {
                label: fromPortal.label,
                icon: fromPortal.icon,
                onSelect: () => {
                  void navigate({ to: fromPortal.to });
                },
              },
            ]
          : []),
        { label: "Sign out", icon: Logout01Icon, onSelect: askSignOut },
      ]}
    />
  );
}
