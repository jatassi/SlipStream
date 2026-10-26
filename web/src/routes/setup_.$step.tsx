import { useQueryClient, useSuspenseQuery } from "@tanstack/react-query";
import { Navigate, createFileRoute, redirect } from "@tanstack/react-router";
import { useEffect } from "react";

import { envOf, placeAt, shownSteps } from "../session/first-run";
import { requireEntry } from "../session/gate";
import { usePasskeySupport } from "../session/passkey";
import {
  firstRunQuery,
  markFirstSetupFinished,
  sessionQuery,
  statusQuery,
} from "../session/session";
import { FrameOverlays } from "./-first-setup/frame";
import { DonePage, StepPage } from "./-first-setup/pages";
import { screens } from "./-first-setup/screens";

export const Route = createFileRoute("/setup_/$step")({
  staticData: { title: "Set up SlipStream" },
  beforeLoad: async ({ context, location, params }) => {
    await requireEntry(context, "first-run", location);
    await context.live.connect();
    const [view, status] = await Promise.all([
      context.queryClient.query({ ...firstRunQuery, staleTime: "static" }),
      context.queryClient.query({ ...statusQuery, staleTime: "static" }),
    ]);
    const at = placeAt(shownSteps(view, envOf(status)), params.step);
    if (at !== params.step) {
      redirect({ to: "/setup/$step", params: { step: at }, replace: true, throw: true });
    }
    return { at };
  },
  loader: async ({ context: { queryClient, at } }) => {
    if (at !== "done") {
      await screens[at].load(queryClient);
    }
  },
  component: FirstSetup,
});

// Another tab or browser finished first setup, and this one follows it into
// the app. The tab that finished marked its session first, so it stays put.
function useFollowFinish(open: boolean) {
  const queryClient = useQueryClient();
  const navigate = Route.useNavigate();
  useEffect(() => {
    if (open || queryClient.getQueryData(sessionQuery.queryKey)?.firstRun !== "open") {
      return;
    }
    markFirstSetupFinished(queryClient);
    void navigate({ to: "/library", replace: true });
  }, [open, queryClient, navigate]);
}

function FirstSetup() {
  const { at } = Route.useRouteContext();
  const view = useSuspenseQuery(firstRunQuery).data;
  const passkeys = usePasskeySupport();
  const shown = shownSteps(view, { passkeys: passkeys.available });
  useFollowFinish(view.open);
  const place = placeAt(shown, at);
  const step = shown.find((candidate) => candidate.id === place);
  return (
    <>
      {place === at ? (
        step === undefined ? (
          <DonePage shown={shown} />
        ) : (
          <StepPage step={step} shown={shown} />
        )
      ) : (
        <Navigate to="/setup/$step" params={{ step: place }} replace />
      )}
      <FrameOverlays passkeys={passkeys.available} />
    </>
  );
}
