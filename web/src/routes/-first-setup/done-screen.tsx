import type { QueryClient } from "@tanstack/react-query";
import { useState } from "react";

import type { FirstrunStepID, FirstrunStepState } from "../../api/gen/model";
import {
  getGetMailSettingsSuspenseQueryOptions,
  useGetMailSettingsSuspense,
} from "../../api/gen/query/mail/mail";
import { freshQuery } from "../../api/query-client";
import { refusalOf } from "../../targets/attempt";
import { AlertLine } from "../../ui/alert-line";
import { Button } from "../../ui/button";
import { Chip } from "../../ui/chip";
import { screens } from "./screens";

/** Reads what the place shows: its step's settings, or the done screen's email settings. */
export async function loadPlace(queryClient: QueryClient, at: FirstrunStepID | "done") {
  await (at === "done"
    ? freshQuery(queryClient, getGetMailSettingsSuspenseQueryOptions())
    : screens[at].load(queryClient));
}

const link =
  "text-meta text-ink-2 underline hover:text-ink-1 in-data-[density=touch]:text-body-touch";

function SettingsLink({ href, onSettings }: { href: string; onSettings: (href: string) => void }) {
  return (
    <a
      href={href}
      className={link}
      onClick={(event) => {
        event.preventDefault();
        onSettings(href);
      }}
    >
      Set it in Settings
    </a>
  );
}

function Row({
  step,
  onSettings,
}: {
  step: FirstrunStepState;
  onSettings: (href: string) => void;
}) {
  const screen = screens[step.id];
  return (
    <li aria-label={screen.name} className="flex flex-wrap items-center gap-x-3 gap-y-1 px-4 py-3">
      <span className="min-w-0 flex-[1_1_12ch] text-ink-1">{screen.name}</span>
      {step.state === "skipped" && (
        <SettingsLink href={screen.settings} onSettings={onSettings} />
      )}
      <Chip look={step.state === "skipped" ? "outline" : "fill"}>
        {step.state === "skipped" ? "Skipped" : "Set"}
      </Chip>
    </li>
  );
}

function EmailLine({ onSettings }: { onSettings: (href: string) => void }) {
  if (useGetMailSettingsSuspense().data.host !== "") {
    return null;
  }
  return (
    <p className="flex flex-wrap items-baseline gap-x-3">
      <span>Email isn&apos;t set up.</span>
      <SettingsLink href="/settings/email" onSettings={onSettings} />
    </p>
  );
}

function Summary({
  shown,
  onSettings,
}: {
  shown: readonly FirstrunStepState[];
  onSettings: (href: string) => void;
}) {
  return (
    <ul
      aria-label="First setup"
      className="divide-y divide-hairline rounded-card bg-glass ring-1 ring-hairline ring-inset"
    >
      {shown.map((step) => (
        <Row key={step.id} step={step} onSettings={onSettings} />
      ))}
    </ul>
  );
}

/** Each shown step as set or skipped. Leaving by any of its ways finishes first setup. */
export function DoneScreen({
  shown,
  back,
  finish,
}: {
  shown: readonly FirstrunStepState[];
  back: (() => void) | undefined;
  finish: (href: string) => Promise<void>;
}) {
  const [leaving, setLeaving] = useState<string>();
  const [failure, setFailure] = useState<string>();
  const leave = (href: string) => {
    setLeaving(href);
    setFailure(undefined);
    finish(href).catch((error: unknown) => {
      setFailure(refusalOf(error));
      setLeaving(undefined);
    });
  };
  return (
    <>
      <p>Each step you skipped can be set later in Settings.</p>
      <Summary shown={shown} onSettings={leave} />
      <EmailLine onSettings={leave} />
      {failure !== undefined && <AlertLine text={failure} />}
      <div className="flex flex-wrap items-center gap-2">
        {back !== undefined && (
          <Button look="ghost" disabled={leaving !== undefined} onClick={back}>
            Back
          </Button>
        )}
        <span className="flex-1" />
        <Button
          look="primary"
          busy={leaving !== undefined}
          onClick={() => {
            leave("/library");
          }}
        >
          Go to Library
        </Button>
      </div>
    </>
  );
}
