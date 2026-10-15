import { Logout01Icon, UserCircleIcon } from "@hugeicons/core-free-icons";
import { Link, useNavigate } from "@tanstack/react-router";
import { type ReactNode, useCallback, useState } from "react";

import { useAccount } from "../../session/session";
import { Menu } from "../../ui/menu";
import { Wordmark } from "../../ui/wordmark";
import { Avatar, useSignOut } from "../account";
import { ConnectionLine } from "../connection-line";
import { portalNav } from "../nav";
import { barGlass } from "../phone/nav-bar";

export function PortalDesktop({ children }: { children: ReactNode }) {
  const [glass, setGlass] = useState(false);
  const sentinel = useCallback((element: HTMLElement) => {
    const observer = new IntersectionObserver(([entry]) => {
      setGlass(entry?.isIntersecting === false);
    });
    observer.observe(element);
    return () => {
      observer.disconnect();
    };
  }, []);
  return (
    <div className="relative [--gutter:clamp(24px,3.6vw,56px)]">
      <span ref={sentinel} aria-hidden="true" className="absolute top-6 left-0 h-px w-px" />
      <TopNav glass={glass} />
      <main className="px-(--gutter) pt-22 pb-30 [view-transition-name:content]">{children}</main>
    </div>
  );
}

function TopNav({ glass }: { glass: boolean }) {
  return (
    <header
      className={`fixed inset-x-0 top-0 z-20 flex h-16 items-center gap-8 px-(--gutter) transition-[background-color,box-shadow] duration-(--duration-control) [view-transition-name:top-bar] ${glass ? barGlass : ""}`}
    >
      <Link to="/portal" aria-label="SlipStream" className="rounded-item text-[19px]">
        <Wordmark />
      </Link>
      <nav aria-label="Portal" className="min-w-0 flex-1">
        <ul className="flex items-center gap-1">
          {portalNav.map((row) => (
            <li key={row.id}>
              <Link
                to={row.to}
                activeOptions={{ exact: true }}
                className="flex h-10 items-center rounded-capsule px-4 text-body-touch text-ink-2 transition-colors duration-(--duration-control) hover:bg-fill hover:text-ink-1 aria-[current=page]:bg-selected aria-[current=page]:font-medium aria-[current=page]:text-ink-1"
              >
                {row.label}
              </Link>
            </li>
          ))}
        </ul>
      </nav>
      <ConnectionLine />
      <AccountMenu />
    </header>
  );
}

function AccountMenu() {
  const { displayName } = useAccount();
  const navigate = useNavigate();
  const signOut = useSignOut();
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
        { label: "Sign out", icon: Logout01Icon, onSelect: signOut },
      ]}
    />
  );
}
