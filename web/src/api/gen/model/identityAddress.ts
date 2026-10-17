export interface IdentityAddress {
  /** The origin links are built on, empty until set */
  address: string;
  /** The address this request reached SlipStream at, or this machine's name when that was an IP address */
  suggestedAddress: string;
}
