/**
 * The probe a Node runs before it is given the Role; absent for none
 */
export type NodesRoleEntryProbe = "engine-starts" | "data-writable" | "listener-up";
