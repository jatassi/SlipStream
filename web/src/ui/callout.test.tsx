import { expect, test } from "vitest";
import { page } from "vitest/browser";

import { renderKit } from "../test/render-kit";
import { Callout } from "./callout";

const today = new Date();
today.setHours(8, 30, 0, 0);

function relay(fixed: string[]) {
  return {
    chip: "Degraded",
    subject: "relay-fra",
    since: today.toISOString(),
    action: {
      label: "Check relay",
      onClick: () => {
        fixed.push("check");
      },
    },
  };
}

test("on desktop a standing warning is a status with its chip, subject, detail, since and one warn action", async () => {
  const fixed: string[] = [];
  renderKit(
    <Callout tone="warn" {...relay(fixed)}>
      Half its probes time out.
    </Callout>,
  );
  const callout = page.getByRole("status");
  await expect
    .element(callout)
    .toHaveTextContent("Degradedrelay-fraHalf its probes time out.since 8:30 AMCheck relay");
  await callout.getByRole("button", { name: "Check relay" }).click();
  expect(fixed).toStrictEqual(["check"]);
});

test("on a phone a standing warning is a row that opens a sheet, and its action closes the sheet first", async () => {
  const fixed: string[] = [];
  renderKit(
    <Callout tone="warn" {...relay(fixed)}>
      Half its probes time out.
    </Callout>,
    { shell: "phone" },
  );
  const row = page.getByRole("button", { name: "Degraded relay-fra" });
  await expect.element(row).toBeVisible();
  await expect.element(page.getByText("Half its probes time out.")).not.toBeInTheDocument();

  await row.click();
  const sheet = page.getByRole("dialog", { name: "relay-fra" });
  await expect.element(sheet.getByText("Half its probes time out.")).toBeVisible();
  await expect.element(sheet.getByText("since 8:30 AM")).toBeVisible();
  await sheet.getByRole("button", { name: "Check relay" }).click();
  expect(fixed).toStrictEqual(["check"]);
  await expect.element(sheet).not.toBeInTheDocument();
});

test("on a phone a condition with no subject shows its sentence in place", async () => {
  renderKit(
    <Callout tone="warn" chip="Slow pattern">
      This pattern took 40ms on 200 releases.
    </Callout>,
    { shell: "phone" },
  );
  await expect.element(page.getByText("This pattern took 40ms on 200 releases.")).toBeVisible();
  await expect.element(page.getByRole("button")).not.toBeInTheDocument();
});

test("a red Callout offers its fix and its other actions", async () => {
  const pressed: string[] = [];
  renderKit(
    <Callout
      tone="fail"
      chip="Not responding"
      subject="Transmission on home"
      action={{
        label: "Check now",
        onClick: () => {
          pressed.push("check");
        },
      }}
      more={[
        {
          label: "Edit client",
          onClick: () => {
            pressed.push("edit");
          },
        },
      ]}
    >
      Transmission isn&apos;t responding.
    </Callout>,
  );
  await page.getByRole("button", { name: "Check now" }).click();
  await page.getByRole("button", { name: "Edit client" }).click();
  expect(pressed).toStrictEqual(["check", "edit"]);
});
