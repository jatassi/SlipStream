import { ArrowRight01Icon } from "@hugeicons/core-free-icons";

import type { RootsSegment } from "../api/gen/model";
import { Button } from "../ui/button";
import { Icon } from "../ui/icon";
import { Menu } from "../ui/menu";
import type { Browser } from "./use-browser";

export function crumbsOf(browser: Browser, nodeName: string): RootsSegment[] {
  return [{ name: nodeName, path: "" }, ...(browser.listing.shown?.segments ?? [])];
}

// Kept whole: the Node and the volume at the start, the last two folders at the end.
const keptAtEnd = 2;
const longCrumbs = 6;

export function Crumbs({ browser, nodeName }: { browser: Browser; nodeName: string }) {
  const crumbs = crumbsOf(browser, nodeName);
  const here = browser.listing.shown?.path ?? null;
  const folded = crumbs.length >= longCrumbs ? crumbs.slice(2, -keptAtEnd) : [];
  const shown =
    folded.length > 0 ? [...crumbs.slice(0, 2), null, ...crumbs.slice(-keptAtEnd)] : crumbs;
  return (
    <nav aria-label="Folders above">
      <ol className="flex min-w-0 items-center gap-0.5 text-body">
        {shown.map((crumb, index) => (
          <li key={crumb?.path ?? "folded"} className="flex min-w-0 items-center gap-0.5">
            {index > 0 && <Icon icon={ArrowRight01Icon} className="size-3.5 shrink-0 text-ink-4" />}
            {crumb === null ? (
              <Menu
                label="More folders above"
                trigger={
                  <Button look="ghost" size="small" aria-label="More folders above">
                    …
                  </Button>
                }
                items={folded.map((hidden) => ({
                  label: hidden.name,
                  onSelect: () => {
                    browser.open(hidden.path);
                  },
                }))}
              />
            ) : crumb.path === here ? (
              <span aria-current="page" className="truncate px-1 font-medium text-ink-1">
                {crumb.name}
              </span>
            ) : (
              <Button
                look="ghost"
                size="small"
                className="max-w-[180px] min-w-0"
                onClick={() => {
                  browser.open(crumb.path);
                }}
              >
                <span className="truncate">{crumb.name}</span>
              </Button>
            )}
          </li>
        ))}
      </ol>
    </nav>
  );
}
