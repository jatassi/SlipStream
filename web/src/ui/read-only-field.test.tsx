import { expect, test, vi } from "vitest";
import { page, userEvent } from "vitest/browser";

import { renderKit } from "../test/render-kit";
import { ReadOnlyField } from "./read-only-field";

const link = "http://forge.localhost:8080/signup#token=Zm9yZ2U";

test("Copy puts the value on the clipboard and says it did", async () => {
  const written = vi.spyOn(navigator.clipboard, "writeText").mockResolvedValue();
  renderKit(<ReadOnlyField label="Signup link" value={link} description="Send it to Grace." />);
  await expect.element(page.getByLabelText("Signup link")).toHaveValue(link);

  await userEvent.click(page.getByRole("button", { name: "Copy Signup link" }));
  await expect.element(page.getByRole("button", { name: "Copied Signup link" })).toBeVisible();
  expect(written.mock.calls).toEqual([[link]]);
  vi.restoreAllMocks();
});
