import { Settings01Icon } from "@hugeicons/core-free-icons";
import { expect, test, vi } from "vitest";
import { page, userEvent } from "vitest/browser";

import { frames } from "../test/frames";
import { renderKit, type Density } from "../test/render-kit";
import { Sheet, SheetCancel, SheetGroup, SheetItem } from "./sheet";

const navRows = ["Nodes", "Settings", "System"];
const ignore = () => {};

function MoreSheet({
  rows = navRows,
  onChoose = ignore,
}: {
  rows?: string[];
  onChoose?: (row: string) => void;
}) {
  return (
    <Sheet trigger={<button type="button">More</button>} label="More">
      <SheetGroup>
        {rows.map((row) => (
          <SheetItem
            key={row}
            icon={Settings01Icon}
            trailing={row === "Nodes" ? <span>3</span> : undefined}
            onClick={() => {
              onChoose(row);
            }}
          >
            {row}
          </SheetItem>
        ))}
      </SheetGroup>
      <SheetCancel />
    </Sheet>
  );
}

async function openMore() {
  await userEvent.click(page.getByRole("button", { name: "More" }));
  const sheet = page.getByRole("dialog", { name: "More" });
  await expect.element(sheet).toBeVisible();
  return sheet;
}

async function touchDrag(target: Element, distance: number) {
  const { left, top, width, height } = target.getBoundingClientRect();
  const at = (y: number) =>
    new Touch({ identifier: 1, target, clientX: left + width / 2, clientY: y });
  const send = (type: string, y: number, touching: boolean) => {
    target.dispatchEvent(
      new TouchEvent(type, {
        bubbles: true,
        cancelable: true,
        touches: touching ? [at(y)] : [],
        changedTouches: [at(y)],
      }),
    );
  };
  const start = top + height / 2;
  send("touchstart", start, true);
  await frames(6, (step) => {
    send("touchmove", start + (distance * step) / 6, true);
  });
  await frames(1);
  send("touchend", start + distance, false);
}

for (const density of ["console", "touch"] satisfies Density[]) {
  test(`opened from a ${density} page, the sheet's rows and Cancel are 52px`, async () => {
    renderKit(<MoreSheet />, { shell: "phone", density });
    const sheet = await openMore();
    const height = (name: string) =>
      getComputedStyle(sheet.getByRole("button", { name }).element()).height;
    expect(height("Nodes 3")).toBe("52px");
    expect(height("Cancel")).toBe("52px");
  });
}

test("the sheet's rows and Cancel set Sentence touch, 17px at 600", async () => {
  renderKit(<MoreSheet />, { shell: "phone" });
  const sheet = await openMore();
  const type = (name: string) => {
    const style = getComputedStyle(sheet.getByRole("button", { name }).element());
    return [style.fontSize, style.fontWeight];
  };
  expect(type("Nodes 3")).toEqual(["17px", "600"]);
  expect(type("Cancel")).toEqual(["17px", "600"]);
});

test("choosing a row runs it and closes the sheet, and Cancel closes it", async () => {
  const onChoose = vi.fn<(row: string) => void>();
  renderKit(<MoreSheet onChoose={onChoose} />, { shell: "phone" });
  let sheet = await openMore();
  await userEvent.click(sheet.getByRole("button", { name: "System" }));
  expect(onChoose).toHaveBeenCalledWith("System");
  await expect.element(sheet).not.toBeInTheDocument();

  sheet = await openMore();
  await userEvent.click(sheet.getByRole("button", { name: "Cancel" }));
  await expect.element(sheet).not.toBeInTheDocument();
  expect(onChoose).toHaveBeenCalledTimes(1);
});

test("the sheet sits on the bottom edge with a 24px top edge", async () => {
  renderKit(<MoreSheet />, { shell: "phone" });
  const sheet = await openMore();
  await expect
    .poll(() => sheet.element().getBoundingClientRect().bottom)
    .toBe(document.documentElement.clientHeight);
  const style = getComputedStyle(sheet.element());
  expect(style.borderTopLeftRadius).toBe("24px");
  expect(style.borderTopRightRadius).toBe("24px");
  expect(style.borderBottomLeftRadius).toBe("0px");
});

test("a sheet taller than the screen scrolls inside the screen", async () => {
  const rows = Array.from({ length: 30 }, (_, index) => `Row ${index + 1}`);
  renderKit(<MoreSheet rows={rows} />, { shell: "phone" });
  const sheet = (await openMore()).element();
  await expect.poll(() => sheet.getBoundingClientRect().top).toBeGreaterThanOrEqual(0);
  expect(sheet.scrollHeight).toBeGreaterThan(sheet.clientHeight);
});

test("a swipe down dismisses the sheet and a short drag springs back", async () => {
  renderKit(<MoreSheet />, { shell: "phone" });
  const sheet = await openMore();
  await frames(25);

  await touchDrag(sheet.getByRole("button", { name: "Settings" }).element(), 20);
  await frames(25);
  await expect.element(sheet).toBeVisible();

  await touchDrag(sheet.getByRole("button", { name: "Settings" }).element(), 250);
  await expect.element(sheet).not.toBeInTheDocument();
});
