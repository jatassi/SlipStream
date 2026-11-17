import { useQueryClient } from "@tanstack/react-query";
import { useState } from "react";

import { sentenceOf } from "../api/errors";
import type { NotificationsBell as Bell, NotificationsEvent as Event } from "../api/gen/model";
import { updateNotificationBell } from "../api/gen/query/notifications/notifications";
import { adoptBell } from "../live/adopt-notifications";
import { Button } from "../ui/button";
import { Switch } from "../ui/switch";
import { EventsChoice } from "./events-field";

/** The bell's events and its warnings switch, saved together. */
export function BellForm({
  bell,
  events,
  onDone,
}: {
  bell: Bell;
  events: readonly Event[];
  onDone: () => void;
}) {
  const queryClient = useQueryClient();
  const [chosen, setChosen] = useState(bell.events);
  const [warnings, setWarnings] = useState(bell.warnings);
  const [busy, setBusy] = useState(false);
  const [refused, setRefused] = useState<string>();
  const save = async () => {
    setBusy(true);
    setRefused(undefined);
    try {
      adoptBell(queryClient, await updateNotificationBell({ events: chosen, warnings }));
      onDone();
    } catch (error) {
      setRefused(sentenceOf(error));
    } finally {
      setBusy(false);
    }
  };
  return (
    <div className="flex flex-col gap-4">
      <p className="text-ink-2">A change applies to what the bell shows from now on.</p>
      <EventsChoice
        events={events}
        chosen={chosen}
        errors={refused === undefined ? [] : [refused]}
        onChange={setChosen}
      />
      <Switch label="Include warnings" checked={warnings} onCheckedChange={setWarnings} />
      <div className="flex flex-wrap justify-end gap-2">
        <Button onClick={onDone}>Cancel</Button>
        <Button look="primary" busy={busy} onClick={() => void save()}>
          Save
        </Button>
      </div>
    </div>
  );
}
