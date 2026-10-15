import { expect, test, vi } from "vitest";
import { page, userEvent } from "vitest/browser";

import { renderKit } from "../test/render-kit";
import { Menu } from "./menu";

const ignore = () => {};

function TorrentMenu({ onSelect = ignore }: { onSelect?: (label: string) => void }) {
  return (
    <Menu
      trigger={<button type="button">Actions</button>}
      label="Torrent actions"
      items={[
        {
          label: "Pause",
          onSelect: () => {
            onSelect("Pause");
          },
        },
        {
          label: "Move up",
          onSelect: () => {
            onSelect("Move up");
          },
        },
        "separator",
        {
          label: "Remove",
          tone: "danger",
          onSelect: () => {
            onSelect("Remove");
          },
        },
      ]}
    />
  );
}

const height = (element: Element) => getComputedStyle(element).height;

test("arrow keys move through the menu and Esc closes it back to the trigger", async () => {
  renderKit(<TorrentMenu />);
  const trigger = page.getByRole("button", { name: "Actions" });
  await userEvent.click(trigger);
  const menu = page.getByRole("menu", { name: "Torrent actions" });
  await expect.element(menu).toBeVisible();

  await userEvent.keyboard("{ArrowDown}");
  await expect.element(page.getByRole("menuitem", { name: "Pause" })).toHaveFocus();
  await userEvent.keyboard("{ArrowDown}");
  await expect.element(page.getByRole("menuitem", { name: "Move up" })).toHaveFocus();
  await userEvent.keyboard("{ArrowDown}");
  await expect.element(page.getByRole("menuitem", { name: "Remove" })).toHaveFocus();
  await userEvent.keyboard("{ArrowDown}");
  await expect.element(page.getByRole("menuitem", { name: "Pause" })).toHaveFocus();
  await userEvent.keyboard("{ArrowUp}");
  await expect.element(page.getByRole("menuitem", { name: "Remove" })).toHaveFocus();

  await userEvent.keyboard("{Escape}");
  await expect.element(menu).not.toBeInTheDocument();
  await expect.element(trigger).toHaveFocus();
});

test("a Console menu's items are 36px and choosing one runs it and closes the menu", async () => {
  const onSelect = vi.fn<(label: string) => void>();
  renderKit(<TorrentMenu onSelect={onSelect} />);
  await userEvent.click(page.getByRole("button", { name: "Actions" }));
  const remove = page.getByRole("menuitem", { name: "Remove" });
  expect(height(remove.element())).toBe("36px");
  await userEvent.click(remove);
  expect(onSelect).toHaveBeenCalledWith("Remove");
  await expect.element(page.getByRole("menu")).not.toBeInTheDocument();
});

test("a desktop menu opened from a Touch page has 52px items", async () => {
  renderKit(<TorrentMenu />, { shell: "desktop", density: "touch" });
  await userEvent.click(page.getByRole("button", { name: "Actions" }));
  await expect.element(page.getByRole("menu", { name: "Torrent actions" })).toBeVisible();
  expect(height(page.getByRole("menuitem", { name: "Pause" }).element())).toBe("52px");
});

test("on phone the menu opens as an action sheet that Cancel closes", async () => {
  renderKit(<TorrentMenu />, { shell: "phone" });
  await userEvent.click(page.getByRole("button", { name: "Actions" }));
  const sheet = page.getByRole("dialog", { name: "Torrent actions" });
  await expect.element(sheet).toBeVisible();
  expect(page.getByRole("menu").query()).toBeNull();
  const pause = sheet.getByRole("button", { name: "Pause" });
  expect(height(pause.element())).toBe("52px");
  expect(height(sheet.getByRole("button", { name: "Cancel" }).element())).toBe("52px");

  await userEvent.click(sheet.getByRole("button", { name: "Cancel" }));
  await expect.element(sheet).not.toBeInTheDocument();
});

test("choosing an action sheet item runs it and closes the sheet", async () => {
  const onSelect = vi.fn<(label: string) => void>();
  renderKit(<TorrentMenu onSelect={onSelect} />, { shell: "phone" });
  await userEvent.click(page.getByRole("button", { name: "Actions" }));
  const sheet = page.getByRole("dialog", { name: "Torrent actions" });
  await userEvent.click(sheet.getByRole("button", { name: "Move up" }));
  expect(onSelect).toHaveBeenCalledWith("Move up");
  await expect.element(sheet).not.toBeInTheDocument();
});

function AccountMenu({ onCheckedChange }: { onCheckedChange: (checked: boolean) => void }) {
  return (
    <Menu
      trigger={<button type="button">Account</button>}
      label="Account"
      items={[{ kind: "toggle", label: "Startup sequence", checked: true, onCheckedChange }]}
    />
  );
}

test("a toggle item is a checked menu checkbox that reports its change", async () => {
  const onCheckedChange = vi.fn<(checked: boolean) => void>();
  renderKit(<AccountMenu onCheckedChange={onCheckedChange} />);
  await userEvent.click(page.getByRole("button", { name: "Account" }));
  const item = page.getByRole("menuitemcheckbox", { name: "Startup sequence" });
  await expect.element(item).toHaveAttribute("aria-checked", "true");
  await userEvent.click(item);
  expect(onCheckedChange).toHaveBeenCalledWith(false);
});

test("on phone a toggle item is a switch row that stays open when changed", async () => {
  const onCheckedChange = vi.fn<(checked: boolean) => void>();
  renderKit(<AccountMenu onCheckedChange={onCheckedChange} />, { shell: "phone" });
  await userEvent.click(page.getByRole("button", { name: "Account" }));
  const sheet = page.getByRole("dialog", { name: "Account" });
  await expect.element(sheet.getByRole("switch", { name: "Startup sequence" })).toBeChecked();
  await userEvent.click(sheet.getByText("Startup sequence"));
  expect(onCheckedChange).toHaveBeenCalledWith(false);
  await expect.element(sheet).toBeVisible();
});
