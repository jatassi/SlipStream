import { createFileRoute } from "@tanstack/react-router";

import { CustomDefinition } from "../../../indexers/add/custom-definition";
import { addStepTrail } from "../../../indexers/add/trail";
import { Page } from "../../../shell/page";

export const Route = createFileRoute("/_app/settings/indexers/new/custom")({
  staticData: { title: "Add a custom definition", trail: addStepTrail },
  component: AddCustomPage,
});

function AddCustomPage() {
  return (
    <Page>
      <CustomDefinition />
    </Page>
  );
}
