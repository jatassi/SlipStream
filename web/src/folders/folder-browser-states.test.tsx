import { expect, test } from "vitest";
import { page, userEvent } from "vitest/browser";

import { readPref } from "../shell/prefs";
import {
  Picker,
  browse,
  current,
  dialog,
  fakeNodes,
  field,
  option,
  place,
  useButton,
} from "../test/folder-browsing";
import { renderQueried, serveFolders } from "../test/folders";
import { asPhone } from "../test/viewport";

test("hidden folders wait behind a counted row and Show hidden, remembered for the next open", async () => {
  await browse({ startPath: "/srv/media" });
  await expect.element(option(".cache")).not.toBeInTheDocument();
  await userEvent.click(option("1 hidden"));
  await expect.element(option(".cache")).toBeVisible();
  await expect.element(page.getByRole("switch", { name: "Show hidden" })).toBeChecked();
  expect(readPref("showHiddenFolders")).toBe(true);
  await userEvent.keyboard("{Escape}");
  await userEvent.click(page.getByRole("button", { name: "Browse" }));
  await expect.element(option(".cache")).toBeVisible();
});

test("a folder SlipStream can't read shows a lock, and opening it names the account under Details", async () => {
  await browse({ startPath: "/srv/media" });
  await userEvent.click(option("Locked Not allowed"));
  await expect
    .element(page.getByText("SlipStream on Media server isn't allowed to open this folder."))
    .toBeVisible();
  await expect.element(page.getByRole("button", { name: "Go up" })).toHaveFocus();
  await userEvent.click(page.getByRole("button", { name: "Details" }));
  await expect
    .element(page.getByText("SlipStream runs as slipstream on Media server.", { exact: false }))
    .toBeVisible();
  await expect.element(page.getByText("open /srv/media/Locked: permission denied")).toBeVisible();
  await userEvent.click(page.getByRole("button", { name: "Go up" }));
  await expect.element(current()).toHaveTextContent("media");
  await expect.element(field()).toHaveFocus();
});

test.each([
  ["/srv/gone", "This folder isn't there."],
  ["/srv/notes.txt", "This is a file, not a folder."],
  [
    "/srv/slow",
    "Media server hasn't answered about this folder. A network share or a sleeping disk can be slow.",
  ],
  ["/srv/broken", "Media server couldn't read this folder."],
])(
  "%s shows its card with Go up and Try again, never an empty folder",
  async (startPath, sentence) => {
    const asked = await browse({ startPath });
    await expect.element(page.getByText(sentence)).toBeVisible();
    await expect.element(page.getByRole("button", { name: "Go up" })).toHaveFocus();
    await expect
      .element(page.getByText("This folder has no folders inside."))
      .not.toBeInTheDocument();
    const reads = asked.filter((path) => path === startPath).length;
    await userEvent.click(page.getByRole("button", { name: "Try again" }));
    await expect.poll(() => asked.filter((path) => path === startPath).length).toBe(reads + 1);
    await userEvent.click(page.getByRole("button", { name: "Go up" }));
    await expect.element(place("/srv/media/Movies")).toBeVisible();
  },
);

test("an offline Node's card says so, with Try again", async () => {
  await browse({ nodeId: "away" });
  await expect
    .element(
      page.getByText("Media server is offline. Its folders can't be browsed until it reconnects."),
    )
    .toBeVisible();
  await expect.element(page.getByRole("button", { name: "Try again" })).toHaveFocus();
  await expect.element(page.getByRole("button", { name: "Go up" })).not.toBeInTheDocument();
});

test("New folder names its row, keeps a refusal in the row, and opens the folder it makes", async () => {
  await browse({ startPath: "/srv/media" });
  await expect.element(option("Anime")).toBeVisible();
  await userEvent.keyboard("{Control>}{Shift>}N{/Shift}{/Control}");
  const name = page.getByRole("textbox", { name: "New folder name" });
  await expect.element(name).toHaveFocus();
  await userEvent.keyboard("Movies{Enter}");
  await expect.element(name).toHaveAccessibleDescription("A folder named Movies is already here.");
  await userEvent.clear(name);
  await userEvent.type(name, "Shows");
  await userEvent.click(page.getByRole("button", { name: "Create" }));
  await expect.element(current()).toHaveTextContent("Shows");
  await expect.element(useButton("Use Shows")).toBeEnabled();
  await expect.element(name).not.toBeInTheDocument();
});

test("the New folder button opens the same row, and Esc puts it away", async () => {
  await browse({ startPath: "/srv/media" });
  await userEvent.click(page.getByRole("button", { name: "New folder" }));
  await expect.element(page.getByRole("textbox", { name: "New folder name" })).toHaveFocus();
  await userEvent.keyboard("{Escape}");
  await expect
    .element(page.getByRole("textbox", { name: "New folder name" }))
    .not.toBeInTheDocument();
  await expect.element(dialog()).toBeVisible();
});

test("on a phone it is a full sheet with a back link named after the parent, and the field waits for a tap", async () => {
  await asPhone();
  serveFolders(fakeNodes(), "away");
  renderQueried(<Picker startPath="/srv/media" />, { shell: "phone" });
  await userEvent.click(page.getByRole("button", { name: "Browse" }));
  await expect.element(page.getByRole("heading", { name: "media" })).toBeVisible();
  await expect.element(page.getByRole("button", { name: "srv" })).toBeVisible();
  await expect
    .element(page.getByRole("button", { name: "/srv/media, folders above" }))
    .toBeVisible();
  await expect.element(field()).not.toHaveFocus();
  expect(option("Anime").element().getBoundingClientRect().height).toBe(54);
  await userEvent.click(page.getByRole("button", { name: "srv" }));
  await expect.element(page.getByRole("heading", { name: "srv" })).toBeVisible();
  await expect.element(useButton("Use srv")).toBeInViewport();
});
