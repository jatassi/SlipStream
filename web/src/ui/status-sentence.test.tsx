import { expect, test } from "vitest";
import { page } from "vitest/browser";

import { renderKit } from "../test/render-kit";

import { StatusSentence } from "./status-sentence";

test("the status sentence reads as one sentence with its links", async () => {
  renderKit(
    <StatusSentence
      parts={[
        { kind: "live", text: "5 jobs moving" },
        { kind: "failure", text: "1 failure", href: "#failures" },
        { kind: "warning", text: "relay-fra degraded", href: "#warnings" },
      ]}
    />,
  );
  const line = page.getByRole("paragraph");
  await expect.element(line).toHaveTextContent("5 jobs moving · 1 failure · relay-fra degraded");

  await expect
    .element(page.getByRole("link", { name: "1 failure" }))
    .toHaveAttribute("href", "#failures");
  await expect
    .element(page.getByRole("link", { name: "relay-fra degraded" }))
    .toHaveAttribute("href", "#warnings");
});
