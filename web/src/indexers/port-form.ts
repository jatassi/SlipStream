import type { IndexersDefinitionForm, IndexersPortFacts } from "../api/gen/model";

/** A ported site's form: a definition form that carries its port's facts. */
export type PortForm = IndexersDefinitionForm & { readonly port: IndexersPortFacts };

/** The form of an entry of origin native, which always carries its port's facts. */
export function portFormOf(form: IndexersDefinitionForm): PortForm {
  const { port } = form;
  if (port === null) {
    throw new Error(`the ported site ${form.entry.id} came without its port's facts`);
  }
  return { ...form, port };
}
