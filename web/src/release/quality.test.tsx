import { expect, test } from "vitest";
import { page, userEvent } from "vitest/browser";

import { renderWithQualities } from "../test/qualities";
import { Quality } from "./quality";

const webdl = "The video is downloaded from a streaming service exactly as the service sends it.";

test("a quality's chip is a button named by its name that opens its source's definition on hover", async () => {
  renderWithQualities(<Quality quality="webdl-1080p" />);
  const chip = page.getByRole("button", { name: "WEBDL-1080p" });
  await expect.element(chip).toHaveAccessibleDescription(webdl);
  await userEvent.hover(chip);
  await expect.element(page.getByRole("dialog", { name: "WEBDL-1080p" })).toHaveTextContent(webdl);
});
