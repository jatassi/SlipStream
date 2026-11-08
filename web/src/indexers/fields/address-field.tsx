import { Select } from "../../ui/select";
import { withFieldGroup } from "../../ui/form";

/** The choice that swaps the known addresses for a typed one; no address is ever this word. */
const typedChoice = "typed";

/** A site's address: one of its known addresses, or one typed when the site moves. */
export type AddressValues = { choice: string; typed: string };

export const startAddress = (links: readonly string[]): AddressValues => ({
  choice: links[0] ?? typedChoice,
  typed: "",
});

/** A saved address: one of the known ones, or typed when the site has moved off them. */
export const savedAddress = (saved: string, links: readonly string[]): AddressValues =>
  links.includes(saved) ? { choice: saved, typed: "" } : { choice: typedChoice, typed: saved };

export const addressOf = ({ choice, typed }: AddressValues) =>
  choice === typedChoice ? typed.trim() : choice;

const anAddress = /^https?:\/\/[^/?#]+(?:\/[^?#]*)?(?:\?[^#]*)?$/u;

function typedError(value: string): string | undefined {
  if (value.trim() === "") {
    return "Type the address, starting with http:// or https://.";
  }
  return anAddress.test(value.trim())
    ? undefined
    : "Type the whole address, starting with http:// or https://.";
}

/**
 * The address of a site: a choice of its known addresses, the first being
 * the default. A ported site can also take a typed one, since a site that
 * moves keeps working before a SlipStream release knows its new address.
 */
export const AddressField = withFieldGroup({
  defaultValues: startAddress([]),
  props: { name: "", links: [] as readonly string[], typeable: true },
  render: function Address({ group, name, links, typeable }) {
    const options = [
      ...links.map((link) => ({ value: link, label: link })),
      ...(typeable ? [{ value: typedChoice, label: "Another address" }] : []),
    ];
    return (
      <div className="flex flex-col gap-3">
        <div className="flex flex-col gap-1.5">
          <group.Field name="choice">
            {(field) => (
              <Select
                label="Address"
                options={options}
                value={field.state.value}
                onValueChange={field.handleChange}
              />
            )}
          </group.Field>
          {typeable && (
            <p className="text-meta text-ink-3 in-data-[density=touch]:text-body-touch">
              Choose an address for {name}, or type a new one if {name} moves.
            </p>
          )}
        </div>
        <group.Subscribe selector={(state) => state.values.choice === typedChoice}>
          {(typing) =>
            typing && (
              <group.AppField
                name="typed"
                validators={{ onSubmit: ({ value }) => typedError(value) }}
              >
                {(field) => (
                  <field.TextField
                    label="New address"
                    type="url"
                    data
                    autoComplete="off"
                    spellCheck={false}
                  />
                )}
              </group.AppField>
            )
          }
        </group.Subscribe>
      </div>
    );
  },
});
