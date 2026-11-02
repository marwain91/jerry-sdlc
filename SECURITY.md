# Security policy

Jerry SDLC is a set of agent instructions. It adds no sandbox, isolation, or attestation: commands run with whatever permissions the host agent has, and separate review passes are not independent in any verified sense, even when the reviewer is another model. A cross-model review sends the diff to the other tool's provider, under that tool's own account and data terms. A Jerry result never authorizes deployment, release, spending, or secret access.

The plugin ships no hooks and no executables. It runs nothing on its own.

The eval suite's `scaffold.sh` scripts run as you, outside the agent sandbox, when you pass `--scaffold`. Read them before you run them.

Please do not report suspected vulnerabilities in a public issue. Use GitHub private vulnerability reporting if it is enabled; otherwise contact the repository owner privately through their GitHub profile. Do not include live credentials, production data, or exploit output containing secrets.

A useful report includes the affected commit, the instruction involved, a minimal reproduction, and the observed impact. No support lifetime or remediation SLA is promised before a stable release.
