import { z } from "zod";

import { makeMediaServerAccount } from "../../api/gen/query/mediaservers/mediaservers";
import { submitErrors, useAppForm } from "../../ui/form";
import { Spinner } from "../../ui/spinner";
import type { BlockProps } from "./plex-block";

const accountForm = z.object({
  username: z.string().trim().min(1, "Enter a username."),
  password: z.string().min(1, "Enter a password."),
});

function MakeAccount({ row }: BlockProps) {
  const form = useAppForm({
    defaultValues: { username: "", password: "" },
    validators: { onSubmit: accountForm },
    onSubmit: async ({ value, formApi }) => {
      await makeMediaServerAccount(row.id, accountForm.parse(value)).catch((error: unknown) => {
        formApi.setErrorMap({
          onSubmit: submitErrors(error, { username_taken: "username", server_refused: "username" }),
        });
      });
    },
  });
  const app = row.server.label;
  return (
    <form
      noValidate
      className="flex flex-col gap-4"
      onSubmit={(event) => {
        event.preventDefault();
        void form.handleSubmit();
      }}
    >
      <form.AppForm>
        <form.AppField name="username">
          {(field) => <field.TextField label="Username" autoComplete="off" />}
        </form.AppField>
        <form.AppField name="password">
          {(field) => (
            <field.TextField
              label="Password"
              type="password"
              autoComplete="new-password"
              description={`Use this password only in the ${app} app. Keep using your PIN for SlipStream.`}
            />
          )}
        </form.AppField>
        <form.FormAlert />
        <form.SubmitButton label={`Make my ${app} account`} />
      </form.AppForm>
      <p className="text-meta text-ink-3 in-data-[density=touch]:text-body-touch">
        Already have an account on {row.server.name}? Ask {row.adminName} to link it.
      </p>
    </form>
  );
}

export function UsersOffered(props: BlockProps) {
  const { row } = props;
  return row.actions.includes("makeAccount") ? (
    <MakeAccount {...props} />
  ) : (
    <p>
      Ask {row.adminName} for an account on {row.server.name}.
    </p>
  );
}

export function UsersMaking({ row }: BlockProps) {
  return (
    <p className="flex items-center gap-2 text-ink-1">
      <Spinner />
      SlipStream is making your {row.server.label} account on {row.server.name}.
    </p>
  );
}
