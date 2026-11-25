export type ReachabilityCheckID =
  | "dns.a"
  | "dns.aaaa"
  | "origin.bind"
  | "origin.firewall"
  | "origin.class"
  | "outside.connect"
  | "outside.connect-back"
  | "tls.certificate"
  | "identity"
  | "http.plain"
  | "plex.custom-url"
  | "plex.mapping";
