# Security policy

Jerry SDLC is pre-release control software and is not yet a trusted release authority. Current local mode uses fully privileged commands, cannot attest worker identity or isolation, audits correction scope after execution, and stores evidence under the same OS owner that can rewrite it. See `PHASE-STATUS.md` before relying on any verdict.

Please do not report suspected vulnerabilities in a public issue. Use GitHub private vulnerability reporting if it is enabled for the repository; otherwise contact the repository owner privately through their GitHub profile. Do not include live credentials, production data, or exploit output containing secrets.

A useful report includes the affected commit, platform, exact trust claim involved, minimal reproduction, observed impact, and whether the behavior crosses a documented boundary or exposes an overclaim. No support lifetime or remediation SLA is promised before the first stable release.
