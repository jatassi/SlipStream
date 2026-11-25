import { useQueryClient } from "@tanstack/react-query";
import { z } from "zod";

import type {
  MediaserversLibraries as Libraries,
  MediaserversLibraryChoice as Library,
  MediaserversPatchBody as Patch,
  MediaserversServerView as Server,
} from "../../api/gen/model";
import {
  updateMediaServer,
  useListMediaServerLibrariesSuspense,
} from "../../api/gen/query/mediaservers/mediaservers";
import { adoptMediaServer } from "../../live/adopt-media-servers";
import { Checkbox } from "../../ui/checkbox";
import { errorText } from "../../ui/form-context";
import { submitErrors, useAppForm } from "../../ui/form";
import { Section } from "../../ui/section";
import { Refusal } from "../../ui/setting-parts";
// oxlint-disable-next-line import/max-dependencies -- the Members part lays out its switch, library checklist, address and Plex Pass line
import { SignalLine } from "../../ui/signal-line";
import { Switch } from "../../ui/switch";
import { toast } from "../../ui/toast";

const membersForm = z
  .object({
    membersSee: z.boolean(),
    libraries: z.array(z.object({ key: z.string(), name: z.string() })),
    address: z.string(),
  })
  .refine((values) => !values.membersSee || values.libraries.length > 0, {
    path: ["libraries"],
    message: "Pick at least one library for Members.",
  });

type Values = z.infer<typeof membersForm>;

const valuesOf = (server: Server): Values => ({
  membersSee: server.membersSee,
  libraries: server.memberLibraries,
  address: server.memberAddress ?? "",
});

const sameLibraries = (a: readonly Library[], b: readonly Library[]) =>
  a.length === b.length && a.every((library) => b.some((other) => other.key === library.key));

function editOf(server: Server, values: Values): Patch {
  const address = values.address.trim();
  return {
    ...(values.membersSee !== server.membersSee && {
      membersSee: values.membersSee,
    }),
    ...(!sameLibraries(values.libraries, server.memberLibraries) && {
      memberLibraries: values.libraries,
    }),
    ...(server.kind !== "plex" &&
      address !== (server.memberAddress ?? "") && {
        memberAddress: address === "" ? null : address,
      }),
  };
}

/** The server's libraries as it lists them now, then any saved choice it no longer lists. */
function choicesOf(live: Libraries, saved: readonly Library[]): Library[] {
  const listed = live.libraries.map(({ key, name }) => ({ key, name }));
  return [
    ...listed,
    ...saved.filter((library) => !listed.some((each) => each.key === library.key)),
  ];
}

/** The libraries behind the shown Path mappings, which Show to Members starts from. */
function managedOf(live: Libraries): Library[] {
  const found = new Map<string, Library>();
  for (const { library } of live.mappings) {
    if (library !== undefined) {
      found.set(library.key, { key: library.key, name: library.name });
    }
  }
  return [...found.values()];
}

const countWords = (count: number) => {
  if (count === 0) {
    return "No Members can watch yet.";
  }
  return count === 1 ? "1 Member can watch." : `${String(count)} Members can watch.`;
};

function useMembersForm(server: Server) {
  const queryClient = useQueryClient();
  return useAppForm({
    defaultValues: valuesOf(server),
    validators: { onSubmit: membersForm },
    onSubmit: async ({ value, formApi }) => {
      const edit = editOf(server, value);
      if (Object.keys(edit).length === 0) {
        return;
      }
      try {
        const got = await updateMediaServer(server.id, edit);
        adoptMediaServer(queryClient, got);
        formApi.reset(valuesOf(got));
        toast(`Saved ${got.name}`);
      } catch (error) {
        formApi.setErrorMap({
          onSubmit: submitErrors(error, { no_member_libraries: "libraries" }),
        });
      }
    },
  });
}

function LibraryChecklist({
  choices,
  picked,
  saved,
  error,
  onPick,
}: {
  choices: readonly Library[];
  picked: readonly Library[];
  saved: readonly Library[];
  error: string | undefined;
  onPick: (picked: Library[]) => void;
}) {
  return (
    <fieldset className="flex flex-col">
      <legend className="mb-1 text-body font-semibold text-ink-2">Libraries for Members</legend>
      {choices.map((library) => (
        <Checkbox
          key={library.key}
          label={library.name}
          checked={picked.some((each) => each.key === library.key)}
          onCheckedChange={(ticked) => {
            onPick(
              ticked ? [...picked, library] : picked.filter((each) => each.key !== library.key),
            );
          }}
        />
      ))}
      {error !== undefined && <Refusal error={error} />}
      {error === undefined && !sameLibraries(picked, saved) && (
        <p className="text-meta text-ink-3">
          Applies to Members given access from now on. People who can already watch keep their
          libraries.
        </p>
      )}
    </fieldset>
  );
}

type MembersForm = ReturnType<typeof useMembersForm>;

function ShowField({ form, live }: { form: MembersForm; live: Libraries }) {
  return (
    <form.Field name="membersSee">
      {(field) => (
        <Switch
          label="Show to Members"
          checked={field.state.value}
          onCheckedChange={(on) => {
            field.handleChange(on);
            if (on && form.getFieldValue("libraries").length === 0) {
              form.setFieldValue("libraries", managedOf(live));
            }
          }}
        />
      )}
    </form.Field>
  );
}

function AddressField({ form, server }: { form: MembersForm; server: Server }) {
  return (
    <form.AppField name="address">
      {(field) => (
        <field.TextField
          label="Address for Members"
          type="url"
          data
          autoComplete="off"
          spellCheck={false}
          description={`The address Members enter in their app to add ${server.name}. Leave it empty and Members are told to ask you for it.`}
        />
      )}
    </form.AppField>
  );
}

/**
 * Whether Members may be offered the server, the libraries their access covers
 * and, on Jellyfin and Emby, the address their app connects to.
 */
export function MembersPart({ server }: { server: Server }) {
  const live = useListMediaServerLibrariesSuspense(server.id).data;
  const form = useMembersForm(server);
  return (
    <Section id="members" title="Members">
      <form
        noValidate
        aria-label={`Members on ${server.name}`}
        className="flex flex-col gap-4"
        onSubmit={(event) => {
          event.preventDefault();
          void form.handleSubmit();
        }}
      >
        <form.AppForm>
          <p className="text-ink-2">{countWords(server.memberCount)}</p>
          <ShowField form={form} live={live} />
          {server.kind === "plex" && server.plexPass === false && (
            <SignalLine tone="warn">
              Your Plex account has no Plex Pass. Members away from home need their own Plex Pass or
              Remote Watch Pass to watch videos.
            </SignalLine>
          )}
          <form.Field name="libraries">
            {(field) => (
              <LibraryChecklist
                choices={choicesOf(live, server.memberLibraries)}
                picked={field.state.value}
                saved={server.memberLibraries}
                error={errorText(field.state.meta.errors)}
                onPick={field.handleChange}
              />
            )}
          </form.Field>
          {server.kind !== "plex" && <AddressField form={form} server={server} />}
          <form.FormAlert />
          <div className="flex justify-end">
            <form.SubmitButton label="Save" />
          </div>
        </form.AppForm>
      </form>
    </Section>
  );
}
