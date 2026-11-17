import { useState } from "react";

import type { NotificationsBell as Bell, NotificationsEvent as Event } from "../api/gen/model";
import {
  useGetNotificationBellSuspense,
  useListNotificationEventsSuspense,
} from "../api/gen/query/notifications/notifications";
import { Button } from "../ui/button";
import { Dialog } from "../ui/dialog";
import { Section } from "../ui/section";
import { BellForm } from "./bell-sheet";

function bellWords(bell: Bell, events: readonly Event[]): string {
  const offered = events.filter((event) => !event.fixed);
  const shown = offered.filter((event) => bell.events.includes(event.id)).length;
  const which = shown === offered.length ? "every event" : `${shown} of ${offered.length} events`;
  return `The bell in the top bar shows ${which}, ${bell.warnings ? "with" : "without"} Health warnings.`;
}

/** The bell's events and warnings switch. */
export function BellSection() {
  const bell = useGetNotificationBellSuspense().data;
  const { events } = useListNotificationEventsSuspense().data;
  const [open, setOpen] = useState(false);
  return (
    <Section
      title="Bell"
      action={
        <Button
          size="small"
          onClick={() => {
            setOpen(true);
          }}
        >
          Choose events
        </Button>
      }
    >
      <p className="mb-3 text-ink-1">{bellWords(bell, events)}</p>
      <Dialog title="The bell's events" open={open} onOpenChange={setOpen}>
        <BellForm
          bell={bell}
          events={events}
          onDone={() => {
            setOpen(false);
          }}
        />
      </Dialog>
    </Section>
  );
}
