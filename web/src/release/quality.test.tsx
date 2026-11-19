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

test("a remux and a CAM open their own source's definition", async () => {
  renderWithQualities(
    <>
      <Quality quality="remux-2160p" />
      <Quality quality="cam" />
    </>,
  );
  await expect
    .element(page.getByRole("button", { name: "Remux-2160p" }))
    .toHaveAccessibleDescription(
      "The video is copied from a Blu-ray disc without encoding it again, so it keeps all of the disc's detail.",
    );
  await expect
    .element(page.getByRole("button", { name: "CAM" }))
    .toHaveAccessibleDescription(
      "The video is recorded in a cinema, often with a handheld camera.",
    );
});

test("a quality as a word sits in its line with a dotted underline", async () => {
  renderWithQualities(
    <p>
      Extended · <Quality quality="bluray-1080p" look="word" />
    </p>,
  );
  const word = page.getByText("Bluray-1080p");
  await expect.element(word).toBeVisible();
  expect(getComputedStyle(word.element()).textDecorationStyle).toBe("dotted");
});
