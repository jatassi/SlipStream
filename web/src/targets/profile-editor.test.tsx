import { HttpResponse, http } from "msw";
import { expect, test } from "vitest";
import { page, userEvent } from "vitest/browser";

import type { TargetsNewProfile, TargetsProfile, TargetsTarget } from "../api/gen/model";
import { getListContentTypesMockHandler } from "../api/gen/query/works/works.msw";
import { CreateProfileBody } from "../api/gen/zod/targets/targets.zod";
import { cinema } from "../test/library";
import { films, profileHandlers, ultra } from "../test/profiles";
import { renderApp } from "../test/render-app";
import { keeper, starter, targetHandlers } from "../test/targets";

function creating() {
  const sent: TargetsNewProfile[] = [];
  const handler = http.post("*/api/v1/profiles", async ({ request }) => {
    const body = CreateProfileBody.parse(await request.json());
    sent.push(body);
    const made: TargetsProfile = {
      ...ultra,
      id: "0199e1f2-7a3b-7c4d-8e5f-00000000000a",
      name: body.name,
    };
    return HttpResponse.json(made, { status: 201 });
  });
  return { sent, handler };
}

const handlers = (
  targets: readonly TargetsTarget[] = [starter, keeper],
  profiles: readonly TargetsProfile[] = [films, ultra],
) => [
  ...targetHandlers(targets),
  ...profileHandlers(profiles),
  getListContentTypesMockHandler({ contentTypes: [cinema] }),
];

const rowsOf = (list: string) =>
  page
    .getByRole("list", { name: list, exact: true })
    .getByRole("listitem")
    .all()
    .map((row) => row.element().textContent);

const sentence = () => page.getByText(/^SlipStream (keeps|always|searches) /u);

async function chooseCuts() {
  await userEvent.click(page.getByRole("radio", { name: "Choose cuts" }));
  await expect.element(sentence()).toHaveTextContent("SlipStream always gets Standard.");
}

test("Choose cuts lists cuts from Standard, says what they want, and warns once Standard is gone", async () => {
  renderApp({ path: "/settings/profiles/new", handlers: handlers() });
  await expect.element(sentence()).toMatchTextContent(/one file at each Target/u);
  await chooseCuts();
  await expect.element(page.getByRole("button", { name: "Remove Standard" })).toBeDisabled();

  await userEvent.click(page.getByRole("button", { name: "Add cut" }));
  await expect
    .poll(() =>
      page
        .getByRole("menuitem")
        .all()
        .map((item) => item.element().textContent),
    )
    .toEqual(["Theatrical", "Extended", "Director's Cut", "IMAX", "Superfan", "Black and Chrome"]);
  await userEvent.click(page.getByRole("menuitem", { name: "Extended" }));
  await expect.element(sentence()).toMatchTextContent(/always gets Standard.+Extended once/u);

  await userEvent.click(page.getByRole("button", { name: "Remove Standard" }));
  expect(rowsOf("Cuts")).toEqual(["Rank 1Extended"]);
  await expect.element(page.getByText("No Standard")).toBeVisible();
  await expect.element(sentence()).toMatchTextContent(/searches for Extended every time/u);
  await expect.element(page.getByRole("button", { name: "Remove Extended" })).toBeDisabled();

  await userEvent.click(page.getByRole("radio", { name: "Any cut" }));
  await expect.element(page.getByText("No Standard")).not.toBeInTheDocument();
  await userEvent.click(page.getByRole("radio", { name: "Choose cuts" }));
  expect(rowsOf("Cuts")).toEqual(["Rank 1Extended"]);
});

test("a saved Profile shows each Target's disk estimate, flags a contained Target, and fixes its type", async () => {
  const small: TargetsTarget = {
    ...keeper,
    id: "0199e1f2-7a3b-7c4d-8e5f-6a7b8c9d0e5d",
    name: "1080p small",
    tiers: [["webdl-1080p"]],
    cutoff: "webdl-1080p",
  };
  const holding = { ...starter, contains: [small.id] };
  const both: TargetsProfile = {
    ...ultra,
    targets: [...ultra.targets, { targetId: small.id, sourcing: "download" }],
  };
  renderApp({
    path: `/settings/profiles/${ultra.id}`,
    handlers: handlers([holding, keeper, small], [films, both]),
  });
  await expect.element(page.getByRole("heading", { name: "Ultra", level: 1 })).toBeVisible();
  await expect.element(page.getByRole("textbox", { name: "Name" })).toHaveValue("Ultra");
  await expect
    .element(page.getByText("For Films. A Profile's Content Type is fixed."))
    .toBeVisible();
  expect(rowsOf("Targets")).toEqual([
    "Rank 12160p HDRNo estimate yet",
    "Rank 21080pAbout 2.1 GB across its Works, at 20 MiB per minute",
    "Rank 31080p smallNo estimate yetEvery file that meets 1080p small also meets 1080p.",
  ]);
  await expect
    .element(page.getByRole("button", { name: "Create Profile" }))
    .not.toBeInTheDocument();
});

test("removing every Target leaves a Profile that wants nothing", async () => {
  renderApp({ path: `/settings/profiles/${films.id}`, handlers: handlers() });
  await userEvent.click(page.getByRole("button", { name: "Remove 1080p" }));
  await expect
    .element(
      page.getByText(
        "This Profile has no Targets. Add one to choose which versions its Works keep.",
      ),
    )
    .toBeVisible();
});

test("a new Profile with chosen cuts creates with them in list order", async () => {
  const { sent, handler } = creating();
  const { router } = renderApp({
    path: "/settings/profiles/new",
    handlers: [...handlers(), handler],
  });
  await userEvent.fill(page.getByRole("textbox", { name: "Name" }), "Cuts");
  await chooseCuts();
  await userEvent.click(page.getByRole("button", { name: "Add cut" }));
  await userEvent.click(page.getByRole("menuitem", { name: "Black and Chrome" }));
  await userEvent.click(page.getByRole("button", { name: "Create Profile" }));
  await expect.poll(() => router.state.location.pathname).toBe("/settings/profiles");
  expect(sent).toEqual([
    {
      name: "Cuts",
      contentType: "cinema",
      targets: [],
      editions: { kind: "cuts", cuts: ["standard", "black-and-chrome"] },
      languages: [],
    },
  ]);
});

test("a saved Profile opens on its chosen cuts", async () => {
  const cuts: TargetsProfile = {
    ...ultra,
    editions: { kind: "cuts", cuts: ["extended", "standard"] },
  };
  renderApp({
    path: `/settings/profiles/${ultra.id}`,
    handlers: handlers(undefined, [films, cuts]),
  });
  await expect.element(page.getByRole("radio", { name: "Choose cuts" })).toBeChecked();
  expect(rowsOf("Cuts")).toEqual(["Rank 1Extended", "Rank 2Standard"]);
});

test("a cut deleted while the Profile was open is said plainly, and nothing is saved", async () => {
  const refusing = http.post("*/api/v1/profiles", () =>
    HttpResponse.json(
      { status: 422, code: "cut_not_found", title: "No cut has that key" },
      { status: 422, headers: { "Content-Type": "application/problem+json" } },
    ),
  );
  const { router } = renderApp({
    path: "/settings/profiles/new",
    handlers: [...handlers(), refusing],
  });
  await userEvent.fill(page.getByRole("textbox", { name: "Name" }), "Cuts");
  await chooseCuts();
  await userEvent.click(page.getByRole("button", { name: "Create Profile" }));
  await expect.element(page.getByRole("alert")).toMatchTextContent(/cut on this list was deleted/u);
  expect(router.state.location.pathname).toBe("/settings/profiles/new");
});
