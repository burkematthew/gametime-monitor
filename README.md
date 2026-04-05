# gametime-monitor

A Go application that monitors youth baseball tournament schedules, scrapes game data from Google Spreadsheets, stores results in PostgreSQL, and automatically creates Google Calendar events. Designed to run on a Raspberry Pi.

## What It Does

1. Scrapes the [GameTime Tournaments](https://www.playgametimestl.com) schedule page for new Google Spreadsheet links
2. Fetches the 10U sheet from each new spreadsheet
3. Parses game entries for a configured team (e.g., "Stl Bears Bell")
4. Stores games in PostgreSQL (with upsert for schedule changes and score updates)
5. Creates Google Calendar events for each game with time, location, and opponent

The app checks every hour on Tuesdays and Wednesdays, since schedules are typically posted mid-week before weekend tournaments.

## Commands

```bash
gametime-monitor serve             # Start the hourly scheduler (Tue/Wed)
gametime-monitor run               # Run a single check and exit
gametime-monitor migrate up        # Apply all pending database migrations
gametime-monitor migrate down      # Revert one migration version
gametime-monitor migrate version   # Show current migration version
```

## Prerequisites

- Docker and Docker Compose (for containerized deployment)
- A Google Cloud service account with the Google Calendar API enabled
- The service account must be shared with your target Google Calendar ("Make changes to events" permission)

### Google Cloud Setup

1. Go to [console.cloud.google.com](https://console.cloud.google.com)
2. Create a project and enable the **Google Calendar API**
3. Create a **Service Account** (APIs & Services > Credentials)
4. Generate a JSON key for the service account
5. In Google Calendar, share your calendar with the service account email and grant **"Make changes to events"** permission

## Configuration

All configuration is done through environment variables. Create a `.env` file from the template:

```bash
cp .env.example .env
```

### Required Variables

| Variable | Description |
|----------|-------------|
| `GCP_KEY_FILE` | Path to the Google service account JSON key file |
| `DB_PASSWORD` | PostgreSQL password |
| `CALENDAR_ID` | Google Calendar ID (found in Calendar Settings) |
| `TEAM_NAME` | Team name to search for in schedules |
| `SCHEDULE_URL` | Tournament schedule page URL |

### Optional Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `DB_HOST` | `localhost` | PostgreSQL host |
| `DB_PORT` | `5432` | PostgreSQL port |
| `DB_USER` | `gametime` | PostgreSQL user |
| `DB_NAME` | `gametime` | PostgreSQL database name |
| `SHEET_NAME` | `10u` | Google Sheets tab name to parse |

## Deployment to Raspberry Pi

### Docker Compose (Recommended)

1. Install Docker on your Pi:
   ```bash
   curl -fsSL https://get.docker.com | sh
   sudo usermod -aG docker $USER
   ```

2. Clone the repo and configure:
   ```bash
   git clone https://github.com/<your-org>/gametime-monitor.git
   cd gametime-monitor
   cp .env.example .env
   # Edit .env with your values
   chmod 600 .env
   ```

3. Place your service account JSON key on the Pi and set `GCP_KEY_FILE` in `.env` to its path.

4. Start the application:
   ```bash
   docker compose up -d
   ```

   This will:
   - Start PostgreSQL
   - Run database migrations automatically
   - Start the scheduler

5. Check logs:
   ```bash
   docker compose logs -f app
   ```

### systemd (Bare Metal)

If running without Docker:

1. Build for your Pi architecture:
   ```bash
   make build-pi64   # ARM64
   # or
   make build-pi     # ARM 32-bit
   ```

2. Copy the binary and service file to the Pi:
   ```bash
   scp gametime-monitor-arm64 pi@<pi-ip>:/home/pi/gametime-monitor/gametime-monitor
   scp gametime-monitor.service pi@<pi-ip>:/tmp/
   scp .env pi@<pi-ip>:/home/pi/gametime-monitor/.env
   ```

3. Install and start the service:
   ```bash
   sudo cp /tmp/gametime-monitor.service /etc/systemd/system/
   sudo systemctl enable --now gametime-monitor
   ```

   The systemd service runs `migrate up` before starting the app via `ExecStartPre`.

## CI/CD

### GitHub Actions CI

Every push to `main` and every pull request runs:

- **Lint** - golangci-lint
- **Test** - full test suite with a PostgreSQL service container
- **Build** - cross-compilation for linux/darwin on amd64/arm64/arm
- **Vulnerability Scan** - govulncheck for Go dependency CVEs
- **Docker Scout** - container image CVE scanning for critical/high severity issues

### Automated Deployment

After CI passes on `main`, the deploy workflow runs on a **self-hosted GitHub Actions runner** on the Pi:

1. Checks out the latest code
2. Writes the GCP service account JSON from a GitHub secret to a temporary file
3. Runs `docker compose up -d --build`
4. Verifies containers are healthy
5. Cleans up the temporary credentials file

The deploy requires manual approval through a GitHub **production environment** with required reviewers.

### Dependabot

Dependabot checks weekly for updates to:

- Go modules
- Docker base images
- GitHub Actions versions

## Self-Hosted Runner Setup

Since this is a public repository, the self-hosted runner requires specific security configuration:

1. **Register the runner** on your Pi:
   ```bash
   mkdir actions-runner && cd actions-runner
   # Download the runner for your architecture from:
   # Settings > Actions > Runners > New self-hosted runner
   ./config.sh --url https://github.com/<your-org>/gametime-monitor --token <TOKEN>
   sudo ./svc.sh install
   sudo ./svc.sh start
   ```

2. **Configure the `production` environment** in GitHub (Settings > Environments):
   - Add required reviewers (yourself)
   - Restrict deployment branches to `main` only

3. **Add secrets** to the `production` environment:
   - `GCP_KEY_JSON` - the full contents of the service account JSON key file
   - `DB_PASSWORD` - PostgreSQL password
   - `CALENDAR_ID` - Google Calendar ID
   - `TEAM_NAME` - team name to filter
   - `SCHEDULE_URL` - tournament schedule URL

4. **Restrict the runner group** (Settings > Actions > Runner groups):
   - Create a runner group for the Pi
   - Set to "Selected workflows only" and select only `deploy.yml`

## Security

### Public Repository Protections

This is a public repository with a self-hosted runner, which requires careful configuration:

- **CI runs only on GitHub-hosted runners** (`ubuntu-latest`) - fork PRs cannot execute code on the Pi
- **Deploy runs only on `self-hosted`** with the `production` environment gate requiring manual approval
- **Runner group restrictions** ensure only the deploy workflow can use the self-hosted runner
- **Fork PR workflows** require approval for outside collaborators (GitHub default for public repos)

### Container Hardening

- **Non-root user** - the app container runs as `appuser`, not root
- **Read-only filesystem** - the app container's root filesystem is mounted read-only
- **no-new-privileges** - prevents privilege escalation inside the container
- **Resource limits** - prevents runaway containers from starving the Pi (128MB/0.25 CPU for app, 256MB/0.5 CPU for Postgres)

### Network Isolation

- **`db` network** - internal only, no external access. PostgreSQL is only reachable by the app container.
- **`egress` network** - allows the app to reach Google APIs and the tournament website
- **No published ports** - PostgreSQL is not exposed to the Pi's LAN or any external network

### Secrets Management

- **Docker secrets** - `DB_PASSWORD` and the GCP service account key are mounted as files at `/run/secrets/`, not passed as environment variables. They are stored in memory (tmpfs), never written to disk in the container, and not visible in `docker inspect` or process listings.
- **GitHub environment secrets** - all sensitive deployment values are stored as secrets on the `production` environment, not as repository-level variables
- **No secrets in logs** - the app does not log connection strings, credentials, or calendar IDs

### Recommendations

- Keep your Pi's OS and Docker installation updated
- Set restrictive permissions on your `.env` file: `chmod 600 .env`
- Grant the GCP service account only `calendar.events` scope on the specific calendar
- Do not run the GitHub Actions runner as root

## Local Development

```bash
# Run tests (requires Docker for database tests)
go test ./pkg/...

# Run a single check locally
GOOGLE_APPLICATION_CREDENTIALS=/path/to/key.json \
DB_PASSWORD=localpass \
CALENDAR_ID=your-calendar-id \
TEAM_NAME="Stl Bears Bell" \
SCHEDULE_URL="https://www.playgametimestl.com/2026-tournament-schedule" \
go run ./cmd/gametime-monitor run

# Build for all platforms
make build         # local
make build-pi64    # Raspberry Pi (64-bit)
make build-pi      # Raspberry Pi (32-bit)
```

## Project Structure

```
cmd/gametime-monitor/main.go       Entry point, config, subcommands, scheduler
pkg/
  model/
    game.go                        Game type, CSV parsing, time resolution
    schedule_link.go               ScheduleLink type
  scraper/
    scraper.go                     HTML scraping, CSV fetching from Google Sheets
  database/
    database.go                    PostgreSQL connection, golang-migrate runner
    models.go                      Bun ORM models (GameRecord, ProcessedSchedule)
    repository.go                  Database queries (upsert, processed checks)
    migrations/                    SQL migration files
  calendar/
    calendar.go                    Google Calendar client, event creation, dedup
```
