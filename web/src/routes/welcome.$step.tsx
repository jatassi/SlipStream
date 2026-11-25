import { createFileRoute, redirect } from "@tanstack/react-router";

import { getListMyMediaServerAccessSuspenseQueryOptions } from "../api/gen/query/mediaservers/mediaservers";
import { ensureQuery } from "../api/query-client";
import { requireEntry } from "../session/gate";
import { passkeySupport } from "../session/passkey";
import { statusQuery } from "../session/session";
import { EntryPage } from "../shell/entry-page";
import { countFrom, nextWelcome, placeWelcome, shownWelcome, welcomeSteps } from "../welcome/steps";
import { StepCount } from "./-links/steps";

export const Route = createFileRoute("/welcome/$step")({
  staticData: { title: "Welcome" },
  beforeLoad: async ({ context, location, params }) => {
    await requireEntry(context, "portal", location);
    await context.live.connect();
    const [status, access] = await Promise.all([
      context.queryClient.query({ ...statusQuery, staleTime: "static" }),
      ensureQuery(context.queryClient, getListMyMediaServerAccessSuspenseQueryOptions()),
    ]);
    const shown = shownWelcome(welcomeSteps, {
      passkeys: passkeySupport(status.passkeys).available,
      servers: access.rows.length,
    });
    const at = placeWelcome(shown, params.step);
    if (at.id !== params.step) {
      redirect({
        to: "/welcome/$step",
        params: { step: at.id },
        replace: true,
        state: location.state,
        throw: true,
      });
    }
    return { shown, at, before: countFrom(location.state) };
  },
  component: Welcome,
});

function Welcome() {
  const { shown, at, before } = Route.useRouteContext();
  const navigate = Route.useNavigate();
  const next = () => {
    void navigate({
      to: "/welcome/$step",
      params: { step: nextWelcome(shown, at).id },
      state: true,
    });
  };
  return (
    <EntryPage title={at.title}>
      {before !== undefined && (
        <StepCount number={before + shown.indexOf(at) + 1} of={before + shown.length} />
      )}
      <at.Step key={at.id} next={next} />
    </EntryPage>
  );
}
