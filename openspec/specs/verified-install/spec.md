# verified-install Specification

## Purpose
The install script only installs an archive whose checksum matches the one the release publishes.

## Requirements

### Requirement: The installer verifies what it downloads
The install script SHALL compare the downloaded archive's SHA-256 with the release's `checksums.txt` and
install only on a match. It SHALL refuse to install when the checksums cannot be fetched, the archive is not
listed, the digest differs, or no tool to compute it is available, unless the user explicitly opts out, in
which case it SHALL warn.

#### Scenario: Genuine release
- **WHEN** the archive matches `checksums.txt`
- **THEN** it reports the verification and installs

#### Scenario: Tampered archive
- **WHEN** the archive's digest differs from the published one
- **THEN** it prints both digests, installs nothing and fails

#### Scenario: No checksums available
- **WHEN** `checksums.txt` cannot be downloaded
- **THEN** it installs nothing, fails, and names the opt-out

#### Scenario: Explicit opt-out
- **WHEN** the user sets the opt-out variable
- **THEN** it warns that verification is skipped and installs
