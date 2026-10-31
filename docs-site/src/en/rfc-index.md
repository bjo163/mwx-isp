# Protocol & RFC Reference

> Bahasa Indonesia: [Versi Indonesia](../id/rfc-index.md)

MWX-ISP implements standard RADIUS, EAP, dynamic-authorization, and
secure-transport protocols. This chapter is the curated, implementation-oriented
index of the standards the project relies on, with each RFC mapped to where it is
used in the code and on the
[roadmap](https://github.com/bjo163/mwx-isp/blob/main/docs/roadmap.md).

The full RFC texts are archived under
[`docs/rfcs/`](https://github.com/bjo163/mwx-isp/tree/main/docs/rfcs);
the raw catalog of every file lives in
[`docs/rfcs/README.md`](https://github.com/bjo163/mwx-isp/blob/main/docs/rfcs/README.md).
Where the two differ, the citations in this chapter take precedence.

## Implemented standards

### RADIUS core

- **RFC 2865** — RADIUS authentication; the base request/response protocol.
- **RFC 2866** — RADIUS accounting; session start / interim-update / stop records.

### RADIUS + EAP integration

- **RFC 3579** — RADIUS support for EAP (EAP-Message and Message-Authenticator).
- **RFC 3580** — IEEE 802.1X RADIUS usage guidelines.

### EAP framework and methods

- **RFC 3748** — Extensible Authentication Protocol (EAP). The framework itself;
  also defines EAP-MD5, the MD5-Challenge method (§5.4).
- **RFC 5216** — EAP-TLS; certificate-based mutual authentication (milestone M1).
- **RFC 5281** — EAP-TTLSv0; a TLS tunnel carrying inner PAP, CHAP, MS-CHAP or
  MS-CHAP-V2 (M9).
- **RFC 1994** — PPP CHAP; the algorithm behind EAP-TTLS inner CHAP. Weak on its
  own and therefore only used inside the TLS tunnel.
- **RFC 2433** — MS-CHAP (v1); the algorithm behind EAP-TTLS inner MS-CHAP. Only
  the NT-Response is accepted; LM-only clients are refused.
- **RFC 2759** — MS-CHAP-V2, used as the inner method of EAP-MSCHAPv2 and PEAPv0
  (M8). MS-MPPE session keys are derived per RFC 2548 / RFC 5705.

> PEAPv0 has no standalone RFC; it follows the Microsoft/Cisco PEAP definition and
> carries inner EAP-MSCHAPv2. It is compatibility-oriented — MS-CHAPv2 exchanges
> carry an NTLMv1-like attack surface — so prefer EAP-TLS where you control client
> certificates.

### Dynamic authorization

- **RFC 5176** — CoA and Disconnect-Request, superseding RFC 3576 (milestone M2).

### Secure transport

- **RFC 6614** — RADIUS over TLS (RadSec).
- **RFC 6613** — RADIUS over TCP.

### Vendor-specific attributes

- **RFC 2548** — Microsoft VSAs, including the MS-MPPE-Send/Recv-Key attributes
  used to carry EAP key material.

### IPv6

- **RFC 3162**, **RFC 4818**, **RFC 6911** — RADIUS IPv6 addressing and
  delegated-prefix attributes (milestone M3).

## Roadmap standards

| RFC | Standard | Milestone | Text archived in `docs/rfcs/` |
| --- | --- | --- | --- |
| RFC 9190 | EAP-TLS 1.3 | M10 | `rfc9190-eap-tls13.txt` |
| RFC 9427 | TLS 1.3 key derivation for TLS-based EAP types | M10, M11 | `rfc9427-tls-based-eap-types-tls13.txt` |
| RFC 9930 | TEAP v1 — tunnel EAP, machine + user chaining; obsoletes RFC 7170 | M11 | `rfc9930-teap-version-1.txt` |
| RFC 7170 | TEAP v1, superseded by RFC 9930 | — | `rfc7170-teap.txt` (history only) |
| RFC 5931 | EAP-PWD — password-based, no per-client certificate | M12 | `rfc5931-eap-pwd.txt` |
| RFC 7542 | Network Access Identifier (NAI) — the anonymous outer identity | M10.5 | `rfc7542-network-access-identifier.txt` |

> **Catalog accuracy note.** **RFC 7542 is the Network Access Identifier (NAI)**,
> whereas **EAP-PWD is RFC 5931**. Until 2026-10-29 the archive held the NAI
> document under the name `rfc7542-eap-pwd.txt` and carried no EAP-PWD text at
> all; RFC 5931 was added and the NAI file was renamed to what it actually is, and
> `internal/eappwdneed` now fails if any file other than `rfc5931-eap-pwd.txt`
> claims the EAP-PWD name. EAP-MD5, on the other hand, is still defined by
> **RFC 3748 §5.4**, not RFC 3851 (an S/MIME specification), so
> `rfc3851-eap-md5.txt` remains a misnamed file the audit reports as
> carried-elsewhere. This chapter uses the correct citations.
> RFC 9930 and RFC 9427 were added to the archive on 2026-10-29 so that the TEAP
> readiness catalog (`internal/teapreadiness`) could stop citing clauses it could
> not read — see [the re-anchoring note](https://github.com/bjo163/mwx-isp/blob/main/docs/teap-citation-anchoring-2026-10-29.md).

## See also

- [Overview](./overview.md) — capability summary, including the EAP suite.
- [Documentation Map](./documentation-map.md) — where every source document lives.
- [RFC Editor](https://www.rfc-editor.org/) — authoritative online RFC texts.
