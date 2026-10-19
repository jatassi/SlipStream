import { ArrowLeft01Icon, FolderAddIcon, MoreHorizontalIcon } from "@hugeicons/core-free-icons";

import { dataFont } from "../ui/data";
import { Icon } from "../ui/icon";
import { IconButton } from "../ui/icon-button";
import { Menu, type MenuEntry } from "../ui/menu";
import { crumbsOf } from "./crumbs";
import type { Parts } from "./parts";

function NavBar({ browser, nodeName, allowNewFolder, close }: Parts) {
  const { current } = browser.listing;
  const inFolder = current !== undefined && current.path !== "";
  const parent = crumbsOf(browser, nodeName).at(-2);
  const more: MenuEntry[] = [
    ...(allowNewFolder && inFolder
      ? [{ label: "New folder", icon: FolderAddIcon, onSelect: browser.newFolder }]
      : []),
    {
      kind: "toggle",
      label: "Show hidden",
      checked: browser.state.showHidden,
      onCheckedChange: browser.toggleHidden,
    },
  ];
  return (
    <div className="flex h-11 items-center gap-2">
      {inFolder && parent !== undefined && (
        <button
          type="button"
          onClick={browser.up}
          className="-ml-1 flex min-w-0 items-center gap-1 rounded-capsule pr-2 text-body-touch text-glow"
        >
          <Icon icon={ArrowLeft01Icon} className="shrink-0" />
          <span className="truncate">{parent.name}</span>
        </button>
      )}
      <span className="flex-1" />
      <Menu
        label="Folder actions"
        trigger={<IconButton icon={MoreHorizontalIcon} label="Folder actions" />}
        items={more}
      />
      <button
        type="button"
        onClick={close}
        className="rounded-capsule px-1 text-body-touch font-semibold text-glow"
      >
        Cancel
      </button>
    </div>
  );
}

/** The folder's path under its name opens its ancestors, and Edit path, on a tap. */
function PathLine({ browser, nodeName }: Parts) {
  const { current } = browser.listing;
  if (current === undefined || current.path === "") {
    return null;
  }
  return (
    <Menu
      label="Folders above"
      trigger={
        <button
          type="button"
          aria-label={`${current.path}, folders above`}
          className={`truncate text-left text-data text-ink-3 ${dataFont}`}
        >
          {current.path}
        </button>
      }
      items={[
        ...crumbsOf(browser, nodeName)
          .slice(0, -1)
          .map((crumb) => ({
            label: crumb.name,
            onSelect: () => {
              browser.open(crumb.path);
            },
          })),
        "separator",
        { label: "Edit path", onSelect: browser.showPath },
      ]}
    />
  );
}

export function PhoneHeader(parts: Parts) {
  return (
    <header className="flex flex-col gap-1">
      <NavBar {...parts} />
      <h2 className="truncate text-title">{parts.browser.name}</h2>
      <PathLine {...parts} />
    </header>
  );
}
