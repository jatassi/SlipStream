import { useQueryClient } from "@tanstack/react-query";
import { useState } from "react";

import type {
  NotificationsConnection as Connection,
  NotificationsEvent as Event,
  NotificationsProvider as Provider,
} from "../api/gen/model";
import {
  addNotificationConnection,
  testNotificationConnection,
  testNotificationDraft,
  updateNotificationConnection,
} from "../api/gen/query/notifications/notifications";
import { adoptConnection } from "../live/adopt-notifications";
import type { TestOutcome } from "../settings/test-line";
import { useAppForm } from "../ui/form";
import { toast } from "../ui/toast";
import {
  type ConnectionValues,
  answerOf,
  draftOf,
  editOf,
  newValues,
  underDetails,
  valuesOf,
} from "./connection-form";

/** What the fields step opened on: a provider to add, or a saved connection. */
export type Opened =
  | { readonly kind: "add"; readonly provider: Provider; readonly firstNode: string }
  | { readonly kind: "edit"; readonly connection: Connection; readonly provider: Provider };

async function save(opened: Opened, value: ConnectionValues) {
  if (opened.kind === "add") {
    const made = await addNotificationConnection(draftOf(value, opened.provider));
    toast(`Added ${made.name}`);
    return made;
  }
  const edit = editOf(opened.connection, value, opened.provider);
  if (Object.keys(edit).length === 0) {
    return null;
  }
  const got = await updateNotificationConnection(opened.connection.id, edit);
  toast(`Saved ${got.name}`);
  return got;
}

const testOf = (opened: Opened, value: ConnectionValues) =>
  opened.kind === "add"
    ? testNotificationDraft(draftOf(value, opened.provider))
    : testNotificationConnection(
        opened.connection.id,
        editOf(opened.connection, value, opened.provider),
      );

/**
 * The connection sheet's form. Test sends a real message from the Node that
 * would send it now, a saved connection's with its saved secrets under any
 * edits. A failed Test or Save shows its sentence and marks the field it
 * names, opening Details when that field sits there.
 */
export function useConnectionSheet(opened: Opened, events: readonly Event[], onDone: () => void) {
  const queryClient = useQueryClient();
  const [outcome, setOutcome] = useState<TestOutcome>();
  const [testing, setTesting] = useState(false);
  const [detailsOpen, setDetailsOpen] = useState(false);
  const { provider } = opened;
  const form = useAppForm({
    defaultValues:
      opened.kind === "add"
        ? newValues(provider, events, opened.firstNode)
        : valuesOf(opened.connection, provider),
    onSubmit: async ({ value }) => {
      setOutcome(undefined);
      try {
        const got = await save(opened, value);
        if (got !== null) {
          adoptConnection(queryClient, got);
        }
      } catch (error) {
        refused(error);
        return;
      }
      onDone();
    },
  });
  function refused(error: unknown) {
    const answer = answerOf(error, provider);
    setOutcome(answer.outcome);
    form.setErrorMap({ onSubmit: answer.errors });
    if (answer.field !== undefined && underDetails(provider, answer.field)) {
      setDetailsOpen(true);
    }
  }
  const test = async () => {
    setTesting(true);
    setOutcome(undefined);
    form.setErrorMap({ onSubmit: { fields: {} } });
    try {
      const { node } = await testOf(opened, form.state.values);
      setOutcome({ passed: true, sentence: `The Test passed. ${node} sent the message.` });
    } catch (error) {
      refused(error);
    } finally {
      setTesting(false);
    }
  };
  return { form, provider, outcome, testing, test, detailsOpen, setDetailsOpen };
}

export type ConnectionSheetState = ReturnType<typeof useConnectionSheet>;
export type ConnectionForm = ConnectionSheetState["form"];
