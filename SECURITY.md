# Security

## What Traverse does

- Reads files under the repository paths given on the command line or in the manifest.
- Writes reports only under `--out`. The command refuses to run if `--out` is inside an analyzed repository or contains one.

## What Traverse never does

| Guarantee | How to verify |
|---|---|
| No network access | No `net`, `net/http` or `crypto/tls` imports (`net/url` is only used to escape paths): `grep -rn '"net' --include=*.go .` |
| No external entities / DTD download (XXE) | XML parsed with `encoding/xml`, which does not resolve external entities |
| No process execution | No `os/exec` imports |
| No modification of analyzed files | Files are only opened with `os.ReadFile` / directory walking; symlinks are not followed |
| No telemetry, no auto-update | No such code exists |
| No third-party code | `go.mod` has no `require` directives |

## Host names

Host names are infrastructure data. Without a key they are never recorded. With a key:

| Aspect | Design |
|---|---|
| Key | 32 random bytes (`traverse keygen`, file mode 0600) or a passphrase (≥ 12 chars, PBKDF2-HMAC-SHA256, 600 000 iterations) |
| Sub-keys | HKDF-SHA256 derives independent pseudonym and encryption keys |
| Pseudonyms | `host-` + first 12 hex chars of HMAC-SHA256(host): stable per key, so systems calling the same server can be correlated without revealing it |
| Vault | `hosts.vault.json` (mode 0600): AES-256-GCM, token as associated data, deterministic nonce derived per host (SIV-like) for reproducible outputs; a key check value detects the wrong key |
| Single exit point | every extractor hands host names to one sealing function; clear text never reaches a report writer |
| Libraries | Go standard library only: `crypto/aes`, `crypto/cipher`, `crypto/hkdf`, `crypto/pbkdf2`, `crypto/hmac`, `crypto/sha256` |

Environment variable and ConfigMap/Secret **values** are never stored; values are only inspected in memory to detect references to other Compose services.

## Vulnerability data and archives

- `--vulndb` reads OSV dumps (`all.zip`, folders of zips or JSON) from disk. No API is queried, so the list of libraries used never leaves the machine. Download the dumps on an authorised machine (`scripts/fetch-osv.sh` / `.ps1`, or a browser) and verify them with `traverse vulndb info`.
- Archives are read in place with `archive/zip`, never extracted to disk or executed:
  - committed JARs: only `META-INF/maven/**/pom.properties` (≤ 64 KiB each);
  - WAR/EAR/JAR deliverables without source code: descriptors, configuration files and embedded library names (≤ 100 000 entries, ≤ 64 MiB per nested archive, ≤ 512 MiB decompressed per artifact);
  - OSV zips: ≤ 16 MiB per record, ≤ 8 GiB in total, ≤ 2 million records.

  Anything over the limits is skipped and reported.
- Keystores and certificates are inspected without decrypting anything: the tool only tells whether a keystore holds a private key and reads public certificates (subject, issuer, expiry). The subject and alias of a natural person's certificate are never stored, and host names inside certificates are sealed like any other host.
- Lockfiles are read as data (JSON, YAML subset, TOML subset, text). Package managers are never invoked.

## Report contents

Reports contain metadata: relative paths, type and package names, HTTP paths, JNDI / queue / table names, library coordinates and line numbers. They never contain source lines, property values, passwords or tokens (secret rules report the location only), URL credentials, full URLs or clear-text host names.

Treat reports as internal information. Use `--classification` to label them.

## Supply chain

- Build with `scripts/build.sh` / `scripts/build.ps1` or the provided CI pipelines: `-trimpath`, `CGO_ENABLED=0`, reproducible output.
- Each release publishes `SHA256SUMS.txt`. Organisations may rebuild from source in their own CI and sign the binaries with their own certificates.

## Reporting a vulnerability

Please report privately to the maintainers (see repository settings → Security) instead of opening a public issue. Include the version (`traverse version`) and a minimal reproduction without confidential code.
