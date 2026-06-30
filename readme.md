> **Deprecation notice:** This project is no longer developed or maintained, and the public hosting infrastructure has been shut down. The tool still works under a **bring-your-own-toolkit (BYOT)** model: provision your own Kafka cluster, Google Cloud Storage bucket, and authority server, then build the CLI with the included Makefile.

<div align="center">
	<img src="./images/logo.png" width="400" height="400" alt="CloudDrop" style="border-radius: 50%;">
	<h1>CloudDrop</h1>
</div>

> A simple CLI tool for sharing files P2P on your local network or over the public internet.

---

## Overview

CloudDrop allows you to share files in two ways:

### 1. Cloud-Based (Public Internet)

Skip the middleman (Discord, Gmail, Slack, etc.) with this CLI tool that doesn't store your data. CloudDrop runs locally on **your** machine and connects to infrastructure you control (Kafka, GCS, and an authority server) for coordination and security.

### 2. P2P (Local Network)

Share files directly with friends or family—both clients just need the CloudDrop CLI installed. Pure byte-to-byte P2P sharing over mDNS on the local network. **No cloud services, Kafka, or authority server required.**

---

## Description

CloudDrop enables you to share files or directories either P2P or over the public internet using Kafka and Google Cloud Storage. It ensures high security via access codes.

---

## Setup (BYOT)

The hosted binary and server are no longer available. Build and configure everything yourself using the paths below.

### Prerequisites

| Requirement | P2P only | Cloud (`send` / `receive`) |
|-------------|----------|----------------------------|
| Go (to build the CLI) | Yes | Yes |
| `make` | Yes | Yes |
| Google Cloud Storage bucket | No | Yes |
| GCS service account (`credentials.json`) | No* | Yes |
| Kafka cluster (e.g. Confluent Cloud) | No | Yes |
| Authority server (Python/FastAPI) | No | Yes |

\*The Makefile reads `authority/credentials.json` at **build time** and embeds it into the binary. For P2P-only use that file is never consulted at runtime, but it must exist for `make install` to succeed—you can use a placeholder `{}` if you will never run `send` or `receive`.

### Path A — P2P only (local network)

Use this if you only need `drop` and `pick` on the same LAN.

1. Clone the repository.
2. Create a placeholder credentials file (only needed for the build step):
   ```bash
   echo '{}' > authority/credentials.json
   ```
3. Edit the Makefile placeholders (`AUTHORITY`, `BUCKET_NAME`, `SECRET`)—values are unused at runtime for P2P, but must be present for the build.
4. Build and install:
   ```bash
   make install
   ```
5. On each machine, run `clouddrop drop /path/to/file` (sender) and `clouddrop pick` (receiver) on the same network.

No `.env` files or authority server are required for P2P.

### Path B — Full cloud stack (public internet)

Use this for `send` and `receive` over the public internet.

#### 1. Google Cloud Storage

- Create a GCS bucket.
- Create a service account with permission to read/write that bucket.
- Save the service account JSON as `authority/credentials.json` (this path is gitignored).

#### 2. Kafka

- Provision a Kafka cluster (Confluent Cloud is what the original deployment used).
- Note the bootstrap server address, API key, and API secret.

#### 3. Authority server

The authority server is a FastAPI app in `authority/` that mediates codes, talks to Kafka, and sweeps expired objects from GCS.

1. Fill in `authority/.env` (committed template with placeholder descriptions—replace with real values locally):
   - `BOOTSTRAP_SERVER`, `KAFKA_API_KEY`, `KAFKA_API_SECRET` — your Kafka credentials
   - `BUCKET_NAME` — same bucket the CLI will use
   - `CREDENTIALS_PATH` — path to your GCS credentials file (default `./credentials.json`, relative to `authority/`)
   - `SECRET` — shared auth token; must match the CLI (see step 4)

2. Install Python dependencies and run:
   ```bash
   cd authority
   python3 -m venv venv
   source venv/bin/activate
   pip install -r requirements.txt
   python main.py
   ```
   The server listens on port `8080`.

   **Docker (optional):** `authority/docker-compose.yml` runs the app behind nginx. The bundled `nginx.conf` contains hardcoded IP/TLS paths from the old hosted deployment—you will need to replace `server_name`, certificate paths, and volume mounts for your own host before use.

#### 4. Build the CLI

Edit the **Makefile** at the repo root (this is the primary configuration path for production builds):

| Makefile variable | Purpose |
|-------------------|---------|
| `AUTHORITY` | Base URL of your authority server (e.g. `https://your-host.example.com`) |
| `BUCKET_NAME` | GCS bucket name |
| `SECRET` | Must match `SECRET` in `authority/.env` |
| `GOOGLE_JSON` | Auto-set from `authority/credentials.json` at build time (base64-encoded) |

Then build and install:

```bash
make install        # build and copy to /usr/local/bin/clouddrop
make release        # build and pack clouddrop-<os>-<arch>.tar.gz
make uninstall      # remove from /usr/local/bin
```

#### 5. Verify

```bash
# Sender
clouddrop send /path/to/file

# Receiver (use the code printed by the sender)
clouddrop receive <code>
```

Both CLI clients must be built against the **same** `AUTHORITY`, `BUCKET_NAME`, and `SECRET`.

### Configuration: Makefile vs `.env`

The repo includes template `.env` files (root and `authority/`) that describe each variable. They are **not** secrets—fill in real values locally only.

| Component | Primary config | Notes |
|-----------|----------------|-------|
| CLI (production) | **Makefile** | Values are embedded at compile time via `-ldflags`. The CLI runs in `PROD` mode by default and uses these embedded defaults for `send`/`receive`. |
| CLI (development) | Root `.env` | Loaded by `godotenv`, but overridden by embedded defaults while `MODE == "PROD"` in `main.go`. Change `MODE` in source to develop against `.env` instead. |
| Authority server | `authority/.env` | Loaded at runtime by Python. `SECRET` must match the Makefile `SECRET`. |

**Important:** `SECRET` in the Makefile and `SECRET` in `authority/.env` must be identical, or authentication between CLI and authority will fail.

### Legacy pre-built binary

Older releases shipped a pre-built `clouddrop.tar.gz` tied to the now-retired hosted infrastructure. That binary will **not** work against a BYOT deployment. Build your own with `make install` or `make release`.

---

## Usage

### Peer-to-Peer (Local Network)

**Client A (Sender):**
```bash
clouddrop drop /path/to/file
```

**Client B (Receiver):**
```bash
clouddrop pick
```

### Over Public Internet

**Client A (Sender):**
```bash
clouddrop send /path/to/file
```

**Client B (Receiver):**
```bash
clouddrop receive <code>
```

---

## Commands

| Command | Description |
|---------|-------------|
| `drop` | Upload files to P2P network. Must provide a valid file path to a file or directory. |
| `pick` | Receive files from P2P network. No arguments required. |
| `send` | Upload files over public internet. Must provide a valid file path. |
| `receive` | Receive files over public internet. Must provide a valid code received from the sender. |

### Limits

| Limit | Value |
|-------|-------|
| Maximum file size (send method) | 5GB |
| File expiry (Public Internet) | 5 minutes after code generation |
| File expiry (P2P) | No expiry (real-time transfer) |

---

## Example Use Cases

- Share files or directories between computers on your local network (P2P)
- Share files over the public internet (requires full BYOT stack)
- Share files on your machine, effectively copying them to the receiver's path

---

## For Future Maintainers

This project is archived. There is no active maintainer and no public hosting.

If you fork or revive the project:

1. Follow the [BYOT setup](#setup-byot) above—nothing works out of the box without your own infrastructure.
2. Never commit real secrets. Template `.env` files in the repo use placeholder descriptions; `credentials.json` is gitignored.
3. Rotate any credentials that were ever used in a local (uncommitted) `.env` before sharing the machine or repo.
4. The authority server's `nginx.conf` and `docker-compose.yml` are reference artifacts from the original deployment—expect to rewrite them for your environment.

Pull requests may still be accepted on a best-effort basis, but there is no guarantee of review or merge.

---

## Environment Variables

### CLI (`.env`)

Reference template at the repo root. Documents the same values embedded via the Makefile for production builds.

| Variable | Description |
|----------|-------------|
| `SECRET` | Secret key for authentication between CLI and authority server |
| `AUTHORITY` | Base URL of the authority server |
| `BUCKET_NAME` | Google Cloud Storage bucket name |
| `GOOGLE_JSON` | Google Cloud service account credentials (JSON string) |

### Authority Server (`authority/.env`)

| Variable | Description |
|----------|-------------|
| `SECRET` | Secret key for authentication (must match CLI `SECRET`) |
| `BUCKET_NAME` | Google Cloud Storage bucket name |
| `CREDENTIALS_PATH` | Path to your Google Cloud `credentials.json` file (relative to `authority/`) |
| `BOOTSTRAP_SERVER` | Kafka bootstrap server address |
| `KAFKA_API_KEY` | Kafka API key |
| `KAFKA_API_SECRET` | Kafka API secret |

---

## License

[MIT](https://choosealicense.com/licenses/mit/)
