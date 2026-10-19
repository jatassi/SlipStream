import { useState } from "react";
import { expect } from "vitest";
import { page, userEvent } from "vitest/browser";

import type { FolderCheck } from "../folders/verdict";
import { FolderBrowser } from "../folders/folder-browser";
import {
  type FakeNode,
  renderQueried,
  serveFolders,
  unixFolder,
  unreadable,
  windowsFolder,
} from "./folders";

const tib = 1024 ** 4;

const media = (): FakeNode => ({
  places: [
    {
      name: "/srv/media/Movies",
      path: "/srv/media/Movies",
      rootId: "7d1f",
      kind: "local",
      free: 1.2 * tib,
      total: 8 * tib,
    },
    { name: "/", path: "/", kind: "local", free: 1.2 * tib, total: 8 * tib },
    { name: "/mnt/nas", path: "/mnt/nas", kind: "network", free: 3 * tib, total: 16 * tib },
  ],
  folders: {
    "/": unixFolder("/", ["mnt", "srv"]),
    "/srv": unixFolder("/srv", ["big", "media"]),
    "/srv/media": unixFolder("/srv/media", [
      "Anime",
      "Movies",
      "TV",
      { name: "Locked", readable: false },
      { name: ".cache", hidden: true },
    ]),
    "/srv/media/Anime": unixFolder("/srv/media/Anime"),
    "/srv/media/TV": unixFolder("/srv/media/TV", ["Frieren"]),
    "/srv/media/TV/Frieren": unixFolder("/srv/media/TV/Frieren"),
    "/srv/media/Movies": unixFolder("/srv/media/Movies"),
    "/srv/media/Locked": unreadable(
      "denied",
      "open /srv/media/Locked: permission denied",
      "slipstream",
    ),
    "/srv/big": unixFolder(
      "/srv/big",
      Array.from({ length: 5000 }, (_, index) => `Folder ${index + 1}`),
    ),
    "/srv/gone": unreadable("not_found", "stat /srv/gone: no such file or directory"),
    "/srv/notes.txt": unreadable("not_a_folder", "open /srv/notes.txt: not a directory"),
    "/srv/slow": unreadable("unresponsive", "stat /srv/slow: no answer within 10s"),
    "/srv/broken": unreadable("io", "readdirent /srv/broken: input/output error"),
  },
  suggests: { "/srv/me": ["/srv/media"] },
});

const windows = (): FakeNode => ({
  places: [{ name: "D: (Media)", path: "D:\\", kind: "local", free: tib, total: 4 * tib }],
  service: true,
  folders: {
    "D:\\": windowsFolder("D:\\", ["Media"]),
    "D:\\Media": windowsFolder("D:\\Media", ["Movies"]),
  },
  cleans: { "d:/media": "D:\\Media" },
  suggests: { "D:\\Me": ["D:\\Media"], "d:/me": ["d:/Media"] },
});

const nameOf = (path: string) => path.split(/[/\\]/u).findLast(Boolean) ?? path;

const libraryRootCheck: FolderCheck = ({ path }) =>
  Promise.resolve(
    path.startsWith("/srv/media/Movies")
      ? {
          usable: false,
          sentence: "Can't use this folder because it's inside the Movies library root.",
        }
      : { usable: true, sentence: `${nameOf(path)} is on /, 1.2 TB free.` },
  );

export function Picker({
  nodeId = "media",
  startPath = "",
}: {
  nodeId?: string;
  startPath?: string;
}) {
  const [open, setOpen] = useState(false);
  const [picked, setPicked] = useState("nothing");
  return (
    <>
      <button
        type="button"
        onClick={() => {
          setOpen(true);
        }}
      >
        Browse
      </button>
      <output aria-label="Picked">{picked}</output>
      <FolderBrowser
        nodeId={nodeId}
        nodeName="Media server"
        open={open}
        onOpenChange={setOpen}
        startPath={startPath}
        check={libraryRootCheck}
        confirmVerb="Use"
        allowNewFolder
        onPick={setPicked}
      />
    </>
  );
}

export const dialog = () => page.getByRole("dialog", { name: "Choose a folder on Media server" });
export const field = () => page.getByRole("combobox", { name: "Filter, or go to a path" });
export const option = (name: string, exact = true) => page.getByRole("option", { name, exact });
/** A place's name runs on into its free space. */
export const place = (name: string) => option(name, false);
export const current = () =>
  page.getByRole("navigation", { name: "Folders above" }).getByRole("listitem").last();
export const useButton = (name: string) => page.getByRole("button", { name, exact: true });

/** A Unix Node, a Windows Node run as a service, and an offline Node, "away". */
export const fakeNodes = () => ({ media: media(), windows: windows() });

export async function browse(props: { nodeId?: string; startPath?: string } = {}) {
  const asked = serveFolders(fakeNodes(), "away");
  renderQueried(<Picker {...props} />);
  await userEvent.click(page.getByRole("button", { name: "Browse" }));
  await expect.element(dialog()).toBeVisible();
  return asked;
}

export async function expectActive(name: string) {
  await expect.element(option(name)).toHaveAttribute("aria-selected", "true");
  expect(field().element().getAttribute("aria-activedescendant")).toBe(option(name).element().id);
}
