export interface ReachabilityGatewayView {
  address: string;
  /** The address the router says it has on the internet, empty when it said none */
  wan?: string;
}
