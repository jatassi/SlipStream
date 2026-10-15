import { ArrowLeft01Icon } from "@hugeicons/core-free-icons";
import { Link } from "@tanstack/react-router";
import type { ReactNode } from "react";

import { Icon } from "../../ui/icon";
import type { Crumb } from "../titles";
import type { PhonePlace } from "./tabs";

export const barGlass =
  "bg-chrome shadow-[inset_0_-1px_0_var(--color-hairline)] backdrop-blur-[24px] backdrop-saturate-[1.6]";

export function NavBar({
  place,
  title,
  back,
  titleShown,
  actions,
}: {
  place: PhonePlace;
  title: string | undefined;
  back: Crumb | undefined;
  titleShown: boolean;
  actions: ReactNode;
}) {
  const solid = place === "push" || titleShown;
  return (
    <header
      className={`sticky top-0 z-20 pt-[env(safe-area-inset-top)] pr-[env(safe-area-inset-right)] pl-[env(safe-area-inset-left)] transition-[background-color,box-shadow] duration-(--duration-control) [view-transition-name:nav-bar] ${solid ? barGlass : ""}`}
    >
      <div className="grid h-11 grid-cols-[1fr_auto_1fr] items-center gap-2 px-2">
        <div className="flex min-w-0">
          {place === "push" && back !== undefined && (
            <Link
              to={back.pathname}
              className="flex h-11 min-w-0 items-center gap-0.5 rounded-item pr-2 text-body-touch text-glow"
            >
              <Icon icon={ArrowLeft01Icon} className="shrink-0" />
              <span className="truncate">{back.title}</span>
            </Link>
          )}
        </div>
        {place === "push" ? (
          <h1 className="max-w-[50vw] truncate text-sentence-touch text-ink-1">{title}</h1>
        ) : (
          <span
            aria-hidden="true"
            className={`max-w-[50vw] truncate text-sentence-touch text-ink-1 transition-opacity duration-(--duration-control) ${titleShown ? "" : "opacity-0"}`}
          >
            {title}
          </span>
        )}
        <div className="flex min-w-0 items-center justify-end gap-1">
          {place === "top" && actions}
        </div>
      </div>
    </header>
  );
}
