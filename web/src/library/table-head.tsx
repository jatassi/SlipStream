import { Checkbox } from "../ui/checkbox";
import type { ColumnDef } from "./columns";
import type { Picking } from "./picking";

export function TableHead({
  columns,
  picking,
  shown,
}: {
  columns: readonly ColumnDef[];
  picking: Picking;
  shown: number;
}) {
  const picked = picking.picked.length - picking.hidden;
  return (
    <thead data-library-head="" className="block">
      <tr aria-rowindex={1} data-library-row="" className="h-8 text-label text-ink-3">
        <th scope="col" data-col="pick">
          <Checkbox
            label="Pick every shown Work"
            hideLabel
            checked={picked > 0 && picked === shown}
            indeterminate={picked > 0 && picked < shown}
            onCheckedChange={picking.toggleAll}
          />
        </th>
        <th scope="col" data-col="work" className="text-left font-medium">
          Title
        </th>
        {columns.map((column) => (
          <th
            key={column.key}
            scope="col"
            data-col={column.key}
            data-align={column.align}
            className="truncate text-left font-medium"
          >
            {column.header}
          </th>
        ))}
      </tr>
    </thead>
  );
}
