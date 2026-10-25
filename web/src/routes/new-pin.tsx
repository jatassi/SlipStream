import { createFileRoute } from "@tanstack/react-router";

import { useChangePIN } from "../api/gen/query/identity/identity";
import { requireEntry } from "../session/gate";
import { pinErrors } from "../session/pin-errors";
import { refreshSession } from "../session/session";
import { EntryPage } from "../shell/entry-page";
import { SignOutDialog, askSignOut } from "../shell/sign-out";
import { Button } from "../ui/button";
import { useAppForm } from "../ui/form";

export const Route = createFileRoute("/new-pin")({
  staticData: { title: "Set a new PIN" },
  beforeLoad: ({ context, location }) => requireEntry(context, "new-pin", location),
  component: NewPin,
});

function useNewPinForm() {
  const { queryClient } = Route.useRouteContext();
  const navigate = Route.useNavigate();
  const changePin = useChangePIN();
  return useAppForm({
    defaultValues: { pin: "" },
    onSubmit: async ({ value, formApi }) => {
      const set = await changePin.mutateAsync({ data: { new: value.pin } }).then(
        () => true,
        (error: unknown) => {
          formApi.setFieldValue("pin", "");
          formApi.setErrorMap({ onSubmit: pinErrors(error) });
          return false;
        },
      );
      if (set) {
        await refreshSession(queryClient);
        await navigate({ to: "/", replace: true });
      }
    },
  });
}

function NewPin() {
  const form = useNewPinForm();
  return (
    <EntryPage title="Set a new PIN">
      <p>Your PIN was turned off after too many wrong PINs. Choose a new one to continue.</p>
      <form
        noValidate
        className="flex flex-col gap-4"
        onSubmit={(event) => {
          event.preventDefault();
          void form.handleSubmit();
        }}
      >
        <form.AppForm>
          <form.AppField name="pin">{(field) => <field.NewPinPad />}</form.AppField>
          <form.FormAlert />
        </form.AppForm>
      </form>
      <Button look="ghost" className="self-start" onClick={askSignOut}>
        Sign out
      </Button>
      <SignOutDialog />
    </EntryPage>
  );
}
