import { expect, test, vi } from "vitest";
import { page, userEvent } from "vitest/browser";

import { type Density, renderKit } from "../test/render-kit";
import { useAppForm } from "./form";

function PinForm({
  onSubmit,
  twice = false,
}: {
  onSubmit: (pin: string) => void;
  twice?: boolean;
}) {
  const form = useAppForm({
    defaultValues: { pin: "" },
    onSubmit: ({ value }) => {
      onSubmit(value.pin);
    },
  });
  return (
    <form
      noValidate
      onSubmit={(event) => {
        event.preventDefault();
        void form.handleSubmit();
      }}
    >
      <form.AppForm>
        <form.AppField name="pin">
          {(field) =>
            twice ? (
              <field.NewPinPad />
            ) : (
              <field.PinPad label="PIN" autoComplete="current-password" autoSubmit />
            )
          }
        </form.AppField>
      </form.AppForm>
    </form>
  );
}

const keypad = () => page.getByRole("group", { name: "Keypad" });

async function press(keys: string) {
  for (const digit of keys) {
    // oxlint-disable-next-line no-await-in-loop -- a person presses one key after another
    await userEvent.click(keypad().getByRole("button", { name: digit, exact: true }));
  }
}

function render(density: Density, twice = false) {
  const submitted = vi.fn<(pin: string) => void>();
  renderKit(<PinForm onSubmit={submitted} twice={twice} />, { density });
  return submitted;
}

test("at the Touch density four key presses fill the PIN and submit it", async () => {
  const submitted = render("touch");
  await press("12");
  await userEvent.click(keypad().getByRole("button", { name: "Delete" }));
  await press("934");
  await expect.poll(() => submitted.mock.calls).toEqual([["1934"]]);
});

test("the keys are 80px circles and the field above keeps the system keyboard down", async () => {
  render("touch");
  await expect.element(keypad()).toBeVisible();
  const key = keypad().getByRole("button", { name: "5" }).element();
  expect(getComputedStyle(key)).toMatchObject({ width: "80px", height: "80px" });
  await expect
    .element(page.getByRole("textbox", { name: "PIN", exact: true }))
    .toHaveAttribute("inputmode", "none");
});

test("at the Console density the PIN is typed and there is no keypad", async () => {
  const submitted = render("console");
  await expect.element(page.getByRole("textbox", { name: "PIN", exact: true })).toBeVisible();
  expect(keypad().query()).toBeNull();
  await userEvent.click(page.getByRole("textbox", { name: "PIN", exact: true }));
  await userEvent.keyboard("4321");
  await expect.poll(() => submitted.mock.calls).toEqual([["4321"]]);
});

test("a new PIN is entered twice, and a mismatch says so and starts over", async () => {
  const submitted = render("touch", true);
  await expect.element(page.getByText("New PIN", { exact: true })).toBeVisible();
  await press("1234");
  await expect.element(page.getByText("Type it again")).toBeVisible();
  await press("1235");
  await expect.element(page.getByText("Those PINs don't match. Choose a new PIN.")).toBeVisible();
  await expect.element(page.getByText("New PIN", { exact: true })).toBeVisible();
  expect(submitted).not.toHaveBeenCalled();

  await press("2468");
  await press("2468");
  await expect.poll(() => submitted.mock.calls).toEqual([["2468"]]);
});

test("at the Console density the second entry is typed straight after the first", async () => {
  const submitted = render("console", true);
  await userEvent.click(page.getByRole("textbox", { name: "New PIN", exact: true }));
  await userEvent.keyboard("1357");
  await expect.element(page.getByText("Type it again")).toBeVisible();
  await userEvent.keyboard("1357");
  await expect.poll(() => submitted.mock.calls).toEqual([["1357"]]);
});
