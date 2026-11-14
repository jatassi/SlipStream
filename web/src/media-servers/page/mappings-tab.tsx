import type { MediaserversServerView as Server } from "../../api/gen/model";
import { useListMediaServerLibrariesSuspense } from "../../api/gen/query/mediaservers/mediaservers";
import { MappingsEditor } from "../mappings-field";

export function MappingsTab({ server }: { server: Server }) {
  const libraries = useListMediaServerLibrariesSuspense(server.id).data;
  return <MappingsEditor server={server} libraries={libraries} />;
}
