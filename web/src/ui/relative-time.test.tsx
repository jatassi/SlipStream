import { expect, onTestFinished, test, vi } from "vitest";
import { page, userEvent } from "vitest/browser";

import { frames } from "../test/frames";
import { renderKit } from "../test/render-kit";
import { PlainTime, RelativeTime } from "./relative-time";

const at = "2026-11-21T14:02:00.000Z";

function useFakeClock(now: string) {
  vi.useFakeTimers({ toFake: ["Date", "setTimeout", "clearTimeout"], now: new Date(now) });
  onTestFinished(() => {
    vi.useRealTimers();
  });
}

const moment = "Saturday, November 21, 2026 at 2:02:00 PM UTC";
const startsWithMoment = new RegExp(`^${moment}`, "u");

test("a time carries its instant and opens its whole moment on hover", async () => {
  renderKit(
    <p className="p-10">
      Since <RelativeTime at={at} form="clock" zone="UTC" />
    </p>,
  );
  const time = page.getByRole("time");
  await expect.element(time).toHaveAttribute("datetime", at);
  await userEvent.hover(page.getByRole("button", { name: /PM$/u }));
  await expect.poll(() => page.getByRole("dialog").query()?.textContent).toMatch(startsWithMoment);
});

test("Enter opens a time's moment from the keyboard", async () => {
  renderKit(<RelativeTime at={at} form="clock" zone="UTC" />);
  await userEvent.tab();
  await userEvent.keyboard("{Enter}");
  await expect.poll(() => page.getByRole("dialog").query()?.textContent).toMatch(startsWithMoment);
});

test("a tap opens a time's moment on a phone", async () => {
  renderKit(<RelativeTime at={at} form="clock" zone="UTC" />, { shell: "phone" });
  await userEvent.click(page.getByRole("button"));
  await expect.poll(() => page.getByRole("dialog").query()?.textContent).toMatch(startsWithMoment);
});

test("a time inside a row is a title and no button", async () => {
  renderKit(
    <a href="#row">
      <PlainTime at={at} form="clock" zone="UTC" />
    </a>,
  );
  await expect
    .element(page.getByRole("time"))
    .toHaveAttribute("title", expect.stringMatching(startsWithMoment));
  await expect.element(page.getByRole("button")).not.toBeInTheDocument();
});

test("an age turns with the shared clock", async () => {
  useFakeClock("2026-11-21T14:02:20.000Z");
  renderKit(<PlainTime at={at} form="age" />);
  await frames(3);
  expect(page.getByRole("time").element().textContent).toBe("just now");
  await vi.advanceTimersByTimeAsync(60_000);
  await frames(3);
  expect(page.getByRole("time").element().textContent).toBe("1 minute ago");
});
