/**
 * Why it can't: an IP address, a name over plain HTTP other than localhost, or a host that is not a domain name
 */
export type SystemPasskeysReason = "ip_address" | "plain_http" | "not_a_domain";
