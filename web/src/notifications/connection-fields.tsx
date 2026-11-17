import type {
  NotificationsConnection as Connection,
  NotificationsEvent as Event,
} from "../api/gen/model";
import { Details } from "../ui/details";
import { Switch } from "../ui/switch";
import { EventsChoice } from "./events-field";
import { type NodeChoice, ProviderField } from "./provider-field";
import type { ConnectionSheetState } from "./use-connection-sheet";

function Fields({
  sheet,
  nodes,
  saved,
}: {
  sheet: ConnectionSheetState;
  nodes: readonly NodeChoice[];
  saved: Connection | undefined;
}) {
  const { form, provider } = sheet;
  const draw = (details: boolean) =>
    provider.fields
      .filter((field) => field.details === details)
      .map((field) => (
        <ProviderField
          key={field.key}
          form={form}
          field={field}
          nodes={nodes}
          sealed={saved?.sealed.includes(field.key) ?? false}
          shown={saved?.shown[field.key]}
        />
      ));
  const details = draw(true);
  return (
    <>
      {draw(false)}
      {details.length > 0 && (
        <Details fields open={sheet.detailsOpen} onOpenChange={sheet.setDetailsOpen}>
          <div className="mt-2 flex flex-col gap-4">{details}</div>
        </Details>
      )}
    </>
  );
}

function Choosing({ sheet, events }: { sheet: ConnectionSheetState; events: readonly Event[] }) {
  const { form, provider } = sheet;
  return (
    <>
      <form.Field name="events">
        {(field) => (
          <EventsChoice
            events={events}
            chosen={field.state.value}
            errors={field.state.meta.errors}
            onChange={field.handleChange}
          />
        )}
      </form.Field>
      <form.Field name="warnings">
        {(field) => (
          <div className="flex flex-col gap-1">
            <Switch
              label="Include warnings"
              checked={field.state.value}
              onCheckedChange={field.handleChange}
            />
            <p className="text-meta text-ink-3">Sends Health warnings as well as errors.</p>
          </div>
        )}
      </form.Field>
      {provider.image !== "none" && (
        <form.Field name="posters">
          {(field) => (
            <div className="flex flex-col gap-1">
              <Switch
                label="Include posters"
                checked={field.state.value}
                onCheckedChange={field.handleChange}
              />
              <p className="text-meta text-ink-3">Adds the poster to each message that has one.</p>
            </div>
          )}
        </form.Field>
      )}
    </>
  );
}

/** The sheet's fields: On, the name, the provider's lead fields, its Details, the events and the switches. */
export function ConnectionFields({
  sheet,
  events,
  nodes,
  saved,
}: {
  sheet: ConnectionSheetState;
  events: readonly Event[];
  nodes: readonly NodeChoice[];
  saved: Connection | undefined;
}) {
  const { form } = sheet;
  return (
    <>
      <form.Field name="on">
        {(field) => (
          <div className="flex justify-end">
            <Switch label="On" checked={field.state.value} onCheckedChange={field.handleChange} />
          </div>
        )}
      </form.Field>
      <form.AppField name="name">
        {(field) => (
          <field.TextField
            label="Name"
            autoComplete="off"
            description="Tells two connections to the same provider apart, such as Family or My phone."
          />
        )}
      </form.AppField>
      <Fields sheet={sheet} nodes={nodes} saved={saved} />
      <Choosing sheet={sheet} events={events} />
    </>
  );
}
