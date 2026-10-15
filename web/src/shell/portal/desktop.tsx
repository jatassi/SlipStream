import { Link } from "@tanstack/react-router";
import { type ReactNode, useCallback, useState } from "react";

import { Wordmark } from "../../ui/wordmark";
import { BrandLink } from "../brand-link";
import { ConnectionLine } from "../connection-line";
import { portalNav } from "../nav";
import { barGlass } from "../phone/nav-bar";
import { AccountMenu } from "./account-menu";

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
      <BrandLink to="/portal" aria-label="SlipStream" className="rounded-item text-[19px]">
        <Wordmark />
      </BrandLink>
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
