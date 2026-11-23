import { expect, test } from "vitest";
import { page, userEvent } from "vitest/browser";

import { accountLists } from "../test/account";
import { signedInAsGrace } from "../test/handlers";
import { asPhone } from "../test/viewport";
import { renderApp } from "../test/render-app";

const mainNav = () => page.getByRole("navigation", { name: "Main" });

test("a Member lands in the portal, under its top nav, at the Touch density", async () => {
  const { router } = renderApp({ path: "/", handlers: [signedInAsGrace] });
  const portal = page.getByRole("navigation", { name: "Portal" });
  await expect
    .element(portal.getByRole("link", { name: "Home" }))
    .toHaveAttribute("aria-current", "page");
  expect(router.state.location.href).toBe("/portal");
  expect(
    portal
      .getByRole("link")
      .all()
      .map((link) => link.element().textContent),
  ).toEqual(["Home", "Search", "Requests", "Account"]);
  await expect.element(page.getByRole("heading", { name: "Home", level: 1 })).toBeVisible();
  await expect.element(mainNav()).not.toBeInTheDocument();
  expect(
    page
      .getByRole("link")
      .all()
      .filter((link) => link.element().getAttribute("aria-current") === "page")
      .map((link) => link.element().textContent),
  ).toEqual(["Home"]);
  expect(document.documentElement.dataset["density"]).toBe("touch");
  expect(document.title).toBe("Home · SlipStream");
});

test("the avatar menu opens Your Account with the Member's name", async () => {
  const { router } = renderApp({ path: "/portal", handlers: [signedInAsGrace, ...accountLists()] });
  await userEvent.click(page.getByRole("button", { name: "Account" }));
  await userEvent.click(page.getByRole("menuitem", { name: "Account" }));
  await expect.poll(() => router.state.location.href).toBe("/portal/account");
  await expect.element(page.getByRole("heading", { name: "Your Account", level: 1 })).toBeVisible();
  await expect
    .element(page.getByRole("region", { name: "Profile" }).getByText("Grace Lindqvist"))
    .toBeVisible();
  await expect.element(page.getByRole("button", { name: "Back to Home" })).not.toBeInTheDocument();

  await userEvent.click(page.getByRole("button", { name: "Account" }));
  await expect
    .poll(() =>
      page
        .getByRole("menuitem")
        .all()
        .map((item) => item.element().textContent),
    )
    .toEqual(["Account", "Sign out"]);
});

test("a Member who opens an Admin link is taken to the portal", async () => {
  const { router } = renderApp({ path: "/activity", handlers: [signedInAsGrace] });
  await expect.poll(() => router.state.location.href).toBe("/portal");
  await expect.element(page.getByRole("heading", { name: "Home", level: 1 })).toBeVisible();
  await expect.element(mainNav()).not.toBeInTheDocument();
});

test("an Admin can open the portal to see what Members see", async () => {
  const { router } = renderApp({ path: "/portal/requests" });
  await expect.element(page.getByRole("heading", { name: "Requests", level: 1 })).toBeVisible();
  expect(router.state.location.href).toBe("/portal/requests");
  await expect
    .element(
      page.getByRole("navigation", { name: "Portal" }).getByRole("link", { name: "Requests" }),
    )
    .toHaveAttribute("aria-current", "page");

  await userEvent.click(page.getByRole("button", { name: "Account" }));
  await userEvent.click(page.getByRole("menuitem", { name: "Back to Home" }));
  await expect.poll(() => router.state.location.href).toBe("/");
  await expect.element(mainNav()).toBeVisible();
});

test("on a phone an Admin finds the way back to Home on the portal's Account page", async () => {
  await asPhone();
  const { router } = renderApp({ path: "/portal/account", handlers: accountLists() });
  await userEvent.click(page.getByRole("button", { name: "Back to Home" }));
  await expect.poll(() => router.state.location.href).toBe("/");
  await expect
    .element(page.getByRole("navigation", { name: "Tabs" }).getByRole("link", { name: "Home" }))
    .toHaveAttribute("aria-current", "page");
});

test("on a phone the portal has its own tab bar and no palette", async () => {
  await asPhone();
  const { router } = renderApp({ path: "/portal", handlers: [signedInAsGrace] });
  const tabs = page.getByRole("navigation", { name: "Tabs" });
  await expect
    .poll(() =>
      tabs
        .getByRole("listitem")
        .all()
        .map((tab) => tab.element().textContent),
    )
    .toEqual(["Home", "Search", "Requests", "Account"]);
  await expect
    .element(tabs.getByRole("link", { name: "Home" }))
    .toHaveAttribute("aria-current", "page");
  await expect.element(page.getByRole("heading", { name: "Home", level: 1 })).toBeVisible();
  await expect.element(page.getByRole("button", { name: "Search" })).not.toBeInTheDocument();
  await expect.element(mainNav()).not.toBeInTheDocument();

  await userEvent.click(tabs.getByRole("link", { name: "Search" }));
  await expect.poll(() => router.state.location.href).toBe("/portal/search");
  await expect
    .element(tabs.getByRole("link", { name: "Search" }))
    .toHaveAttribute("aria-current", "page");
  await expect
    .element(tabs.getByRole("link", { name: "Home" }))
    .not.toHaveAttribute("aria-current");
});
