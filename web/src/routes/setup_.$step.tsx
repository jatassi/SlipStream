import { useQueryClient, useSuspenseQuery } from "@tanstack/react-query";
import { Navigate, createFileRoute, redirect } from "@tanstack/react-router";
import { useEffect, useRef } from "react";

import { envOf, placeAt, shownSteps } from "../session/first-run";
import { requireEntry } from "../session/gate";
import { usePasskeySupport } from "../session/passkey";
import {
  finishFirstSetup,
  firstRunQuery,
  markFirstSetupFinished,
  sessionQuery,
  statusQuery,
} from "../session/session";
import { FrameOverlays } from "./-first-setup/frame";
import { loadPlace } from "./-first-setup/done-screen";
import { DonePage, StepPage } from "./-first-setup/pages";

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
  loader: ({ context: { queryClient, at } }) => loadPlace(queryClient, at),
  component: FirstSetup,
});

/**
 * Follows a finish from another tab or browser into the Library, and returns
 * this tab's own way out: finish, then go to `href`. While this tab finishes,
 * the firstrun.changed its finish sends doesn't move it.
 */
function useFinish(open: boolean) {
  const queryClient = useQueryClient();
  const navigate = Route.useNavigate();
  const leaving = useRef(false);
  useEffect(() => {
    if (
      open ||
      leaving.current ||
      queryClient.getQueryData(sessionQuery.queryKey)?.firstRun !== "open"
    ) {
      return;
    }
    markFirstSetupFinished(queryClient);
    void navigate({ to: "/library", replace: true });
  }, [open, queryClient, navigate]);
  return async (href: string) => {
    leaving.current = true;
    try {
      await finishFirstSetup(queryClient);
    } finally {
      leaving.current = false;
    }
    await navigate({ href });
  };
}

function FirstSetup() {
  const { at } = Route.useRouteContext();
  const view = useSuspenseQuery(firstRunQuery).data;
  const passkeys = usePasskeySupport();
  const shown = shownSteps(view, { passkeys: passkeys.available });
  const leave = useFinish(view.open);
  const place = placeAt(shown, at);
  const step = shown.find((candidate) => candidate.id === place);
  return (
    <>
      {place === at ? (
        step === undefined ? (
          <DonePage shown={shown} leave={leave} />
        ) : (
          <StepPage step={step} shown={shown} leave={leave} />
        )
      ) : (
        <Navigate to="/setup/$step" params={{ step: place }} replace />
      )}
      <FrameOverlays passkeys={passkeys.available} />
    </>
  );
}
