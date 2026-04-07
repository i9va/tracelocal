# Security Policy

## Scope

tracelocal is a local development tool. It binds to `localhost` by default and is not designed to be exposed to the public internet. That said, security issues in the binary itself — such as malformed OTLP payloads causing crashes or memory corruption — are in scope.

**Out of scope:**
- Issues that require the tool to be intentionally exposed to untrusted networks
- Denial-of-service via high span volume (expected behaviour; use `--capacity` to limit memory)

## Supported versions

Only the latest release receives security fixes.

| Version | Supported |
|---------|-----------|
| Latest  | Yes       |
| Older   | No        |

## Reporting a vulnerability

**Do not open a public GitHub issue for security vulnerabilities.**

Please report security issues by emailing the maintainer directly. You can find contact details on the [GitHub profile](https://github.com/henriqueholanda). Include:

- A description of the vulnerability and its potential impact
- Steps to reproduce or a minimal proof of concept
- Any suggested mitigations, if you have them

You can expect an acknowledgement within 48 hours and a fix or status update within 14 days. We'll credit you in the release notes unless you prefer to remain anonymous.
