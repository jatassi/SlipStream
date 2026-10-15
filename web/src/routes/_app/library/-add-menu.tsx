import { Add01Icon } from "@hugeicons/core-free-icons";
import { useNavigate } from "@tanstack/react-router";

import type { WorksContentType } from "../../../api/gen/model";
import { Button } from "../../../ui/button";
import { Menu } from "../../../ui/menu";
import { addLabel } from "../../../works/fields";

export function AddMenu({ types }: { types: readonly WorksContentType[] }) {
  const navigate = useNavigate();
  return (
    <Menu
      label="Add a Work"
      trigger={
        <Button look="primary" icon={Add01Icon}>
          Add
        </Button>
      }
      items={types.map((type) => ({
        label: addLabel(type),
        onSelect: () => {
          void navigate({ to: "/add", search: { contentType: type.id } });
        },
      }))}
    />
  );
}
