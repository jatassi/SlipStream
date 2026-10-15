import { expect, test } from "vitest";
import { page, userEvent } from "vitest/browser";

import { renderKit } from "../test/render-kit";
import { Dialog, DialogClose, type DialogProps } from "./dialog";

function RemoveDialog({ size }: Pick<DialogProps, "size">) {
  return (
    <Dialog
      title="Remove torrents"
      trigger={<button type="button">Remove…</button>}
      {...(size && { size })}
    >
      <label>
        Reason <input />
      </label>
      <DialogClose>
        <button type="button">Cancel</button>
      </DialogClose>
      <button type="button">Remove</button>
    </Dialog>
  );
}

const style = (name: string) => getComputedStyle(page.getByRole("dialog", { name }).element());

test("focus stays inside the dialog and returns to its opener on Esc", async () => {
  renderKit(<RemoveDialog />);
  const opener = page.getByRole("button", { name: "Remove…" });
  await userEvent.click(opener);
  const dialog = page.getByRole("dialog", { name: "Remove torrents" });
  await expect.element(dialog).toBeVisible();

  const reason = dialog.getByRole("textbox", { name: "Reason" });
  await expect.element(reason).toHaveFocus();
  await userEvent.tab();
  await expect.element(dialog.getByRole("button", { name: "Cancel" })).toHaveFocus();
  await userEvent.tab();
  await expect.element(dialog.getByRole("button", { name: "Remove" })).toHaveFocus();
  await userEvent.tab();
  await expect.element(reason).toHaveFocus();
  await userEvent.tab({ shift: true });
  await expect.element(dialog.getByRole("button", { name: "Remove" })).toHaveFocus();

  await userEvent.keyboard("{Escape}");
  await expect.element(dialog).not.toBeInTheDocument();
  await expect.element(opener).toHaveFocus();
});

test("a close button inside the dialog closes it and returns focus", async () => {
  renderKit(<RemoveDialog />);
  const opener = page.getByRole("button", { name: "Remove…" });
  await userEvent.click(opener);
  await userEvent.click(page.getByRole("button", { name: "Cancel" }));
  await expect.element(page.getByRole("dialog")).not.toBeInTheDocument();
  await expect.element(opener).toHaveFocus();
});

test("a desktop dialog is 460px wide by default and 940px at its large size, radius 20", async () => {
  renderKit(<RemoveDialog />);
  await userEvent.click(page.getByRole("button", { name: "Remove…" }));
  await expect.element(page.getByRole("dialog", { name: "Remove torrents" })).toBeVisible();
  expect(style("Remove torrents").width).toBe("460px");
  expect(style("Remove torrents").borderTopLeftRadius).toBe("20px");
  await userEvent.keyboard("{Escape}");
  await expect.element(page.getByRole("dialog")).not.toBeInTheDocument();

  renderKit(<RemoveDialog size={940} />, { density: "touch" });
  await userEvent.click(page.getByRole("button", { name: "Remove…" }).last());
  await expect.element(page.getByRole("dialog", { name: "Remove torrents" })).toBeVisible();
  expect(style("Remove torrents").width).toBe("940px");
});

test("on phone the dialog is a bottom sheet that traps focus and returns it", async () => {
  renderKit(<RemoveDialog />, { shell: "phone" });
  const opener = page.getByRole("button", { name: "Remove…" });
  await userEvent.click(opener);
  const sheet = page.getByRole("dialog", { name: "Remove torrents" });
  await expect.element(sheet).toBeVisible();
  await expect
    .poll(() => sheet.element().getBoundingClientRect().bottom)
    .toBe(document.documentElement.clientHeight);
  expect(style("Remove torrents").borderTopLeftRadius).toBe("24px");
  expect(style("Remove torrents").borderBottomLeftRadius).toBe("0px");

  const reason = sheet.getByRole("textbox", { name: "Reason" });
  await userEvent.click(reason);
  await userEvent.tab({ shift: true });
  await expect.element(sheet.getByRole("button", { name: "Remove" })).toHaveFocus();

  await userEvent.keyboard("{Escape}");
  await expect.element(sheet).not.toBeInTheDocument();
  await expect.element(opener).toHaveFocus();
});
