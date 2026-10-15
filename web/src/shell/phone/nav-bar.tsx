import { ArrowLeft01Icon, Search01Icon } from "@hugeicons/core-free-icons";
import { Link } from "@tanstack/react-router";

import { useAccount } from "../../session/session";
import { Icon } from "../../ui/icon";
import { IconButton } from "../../ui/icon-button";
import { Avatar } from "../account";
import { openPalette } from "../palette-state";
import type { Crumb } from "../titles";
import { openMore } from "./more-state";
import type { PhonePlace } from "./tabs";

const glass =
  "bg-chrome shadow-[inset_0_-1px_0_var(--color-hairline)] backdrop-blur-[24px] backdrop-saturate-[1.6]";

export function NavBar({
  place,
  title,
  back,
  titleShown,
}: {
  place: PhonePlace;
  title: string | undefined;
  back: Crumb | undefined;
  titleShown: boolean;
}) {
  const solid = place === "push" || titleShown;
  return (
    <header
      className={`sticky top-0 z-20 pt-[env(safe-area-inset-top)] pr-[env(safe-area-inset-right)] pl-[env(safe-area-inset-left)] transition-[background-color,box-shadow] duration-(--duration-control) [view-transition-name:nav-bar] ${solid ? glass : ""}`}
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
          {place === "top" && <TopActions />}
        </div>
      </div>
    </header>
  );
}

function TopActions() {
  const { displayName } = useAccount();
  return (
    <>
      <IconButton icon={Search01Icon} label="Search" onClick={openPalette} />
      <button
        type="button"
        aria-label="Account"
        onClick={openMore}
        className="grid size-11 place-items-center rounded-capsule"
      >
        <Avatar name={displayName} />
      </button>
    </>
  );
}
