import type { ShellAlertsConversions } from "./shellAlertsConversions.ts";
import type { ShellAlertsHealth } from "./shellAlertsHealth.ts";
import type { ShellAlertsNodes } from "./shellAlertsNodes.ts";
import type { ShellAlertsTasks } from "./shellAlertsTasks.ts";

export interface ShellAlerts {
  /**
   * fail while a Conversion failed and waits for an Admin, warn while no Node can convert one; null otherwise
   * @nullable
   */
  conversions: ShellAlertsConversions;
  /**
   * The worst level across every Health category, null while nothing is wrong
   * @nullable
   */
  health: ShellAlertsHealth;
  /**
   * warn while a Node has been offline longer than a brief drop, or the Controller is doing a Role's work for an offline Node; fail while a Node has been offline or on another version longer than the offline grace; null otherwise
   * @nullable
   */
  nodes: ShellAlertsNodes;
  /**
   * The worst level in the Scheduled tasks Health category, null while nothing is wrong
   * @nullable
   */
  tasks: ShellAlertsTasks;
}
