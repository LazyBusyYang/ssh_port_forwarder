# Local Docker Test Environment

`docker-compose.local-test.yml` starts a local-only stack for UI/API validation:

- frontend Vite dev server
- backend Go service
- MySQL 8
- SSH host with password auth
- SSH host with key auth

All credentials, passwords, JWT/encryption values, and SSH keys documented here are test fixtures only. Do not reuse them in production or personal infrastructure.

## Start, stop, status, logs

```bash
make local-test-up       # start all services in the background
make local-test-ps       # show service status
make local-test-logs     # follow logs
make local-test-down     # stop containers, keep named volumes
make local-test-clean    # stop containers and remove local-test volumes
make local-test-config   # render/validate compose config without starting services
```

Equivalent Compose command:

```bash
docker compose -f docker-compose.local-test.yml -p spf-local-test ps
```

## Default ports and URLs

| Component | Host URL / port | Compose service endpoint |
| --- | --- | --- |
| Frontend dev UI | <http://localhost:5173> | `frontend` shares `backend` network namespace |
| Backend API | <http://localhost:8080> | `backend:8080` |
| MySQL | `127.0.0.1:3307` | `mysql:3306` |
| Password SSH fixture | `127.0.0.1:2222` | `ssh-password:2222` |
| Key SSH fixture | `127.0.0.1:2223` | `ssh-key:2222` |

The frontend service intentionally shares the backend network namespace so the existing Vite proxy target `http://localhost:8080` works without changing `web/vite.config.ts`.

## Application login

The backend creates/verifies the default admin from local-only env fixtures:

- Username: `admin`
- Password: `admin123`

## SSH password fixture

Use these values when creating an SSH Host inside the Dockerized app:

- Host: `ssh-password`
- Port: `2222`
- Username: `testuser`
- Auth method: password
- Password: `testpass123`

From the host machine, quick connectivity check after `make local-test-up`:

```bash
ssh -p 2222 -o StrictHostKeyChecking=no testuser@127.0.0.1 true
# password: testpass123
```

## SSH key fixture

Use these values when creating an SSH Host inside the Dockerized app:

- Host: `ssh-key`
- Port: `2222`
- Username: `keyuser`
- Auth method: private key
- Private key: the local-only fixture below

```text
-----BEGIN OPENSSH PRIVATE KEY-----
b3BlbnNzaC1rZXktdjEAAAAABG5vbmUAAAAEbm9uZQAAAAAAAAABAAAAMwAAAAtzc2gtZW
QyNTUxOQAAACB3TrSG3OBvDyEbyf+IXbMnPoFX7QUvtNZzcO9UOus+SgAAAKg5LEhTOSxI
UwAAAAtzc2gtZWQyNTUxOQAAACB3TrSG3OBvDyEbyf+IXbMnPoFX7QUvtNZzcO9UOus+Sg
AAAECFrTqN9xpFgQODXJ1GVDG/0rTHDKFnW/XiOLZIzw3k0XdOtIbc4G8PIRvJ/4hdsyc+
gVftBS+01nNw71Q66z5KAAAAJXNwZi1sb2NhbC10ZXN0LWtleS1ub3QtZm9yLXByb2R1Y3
Rpb24=
-----END OPENSSH PRIVATE KEY-----
```

From the host machine, save the fixture and test it:

```bash
cat > /tmp/spf-local-test-key <<'KEY'
-----BEGIN OPENSSH PRIVATE KEY-----
b3BlbnNzaC1rZXktdjEAAAAABG5vbmUAAAAEbm9uZQAAAAAAAAABAAAAMwAAAAtzc2gtZW
QyNTUxOQAAACB3TrSG3OBvDyEbyf+IXbMnPoFX7QUvtNZzcO9UOus+SgAAAKg5LEhTOSxI
UwAAAAtzc2gtZWQyNTUxOQAAACB3TrSG3OBvDyEbyf+IXbMnPoFX7QUvtNZzcO9UOus+Sg
AAAECFrTqN9xpFgQODXJ1GVDG/0rTHDKFnW/XiOLZIzw3k0XdOtIbc4G8PIRvJ/4hdsyc+
gVftBS+01nNw71Q66z5KAAAAJXNwZi1sb2NhbC10ZXN0LWtleS1ub3QtZm9yLXByb2R1Y3
Rpb24=
-----END OPENSSH PRIVATE KEY-----
KEY
chmod 600 /tmp/spf-local-test-key
ssh -i /tmp/spf-local-test-key -p 2223 -o StrictHostKeyChecking=no keyuser@127.0.0.1 true
```

The matching public key is embedded in `docker-compose.local-test.yml` as `ssh-key.environment.PUBLIC_KEY`.

## MySQL connection

- Host from your machine: `127.0.0.1`
- Port from your machine: `3307`
- Compose service host: `mysql`
- Database: `spf_local_test`
- User: `root`
- Password: `spf_local_test_root`

The backend uses the DSN fixture:

```text
root:spf_local_test_root@tcp(mysql:3306)/spf_local_test?charset=utf8mb4&parseTime=true&loc=Local
```

## Cleanup

Use `make local-test-down` when you want to keep database/cache volumes for another run.

Use `make local-test-clean` when you want a fresh local test environment. This removes Compose named volumes for MySQL data, Go caches, frontend `node_modules`, and SSH server config.

## Troubleshooting

- **Port already in use**: stop the conflicting local service or edit the host-side port mapping in `docker-compose.local-test.yml` (`8080`, `5173`, `3307`, `2222`, `2223`).
- **Backend cannot connect to SSH fixtures**: when using the Dockerized backend, create SSH Hosts with `ssh-password:2222` or `ssh-key:2222`, not `127.0.0.1`.
- **Host machine SSH check fails with key permissions**: run `chmod 600 /tmp/spf-local-test-key`.
- **Frontend API calls fail**: confirm both `backend` and `frontend` are running with `make local-test-ps`; the frontend relies on sharing the backend network namespace for Vite proxying.
- **Need a clean database**: run `make local-test-clean` and then `make local-test-up`.
- **Image pull issues**: retry after network recovers; the stack uses public images `mysql:8.0`, `golang:1.25-alpine`, `node:22-alpine`, and `lscr.io/linuxserver/openssh-server:latest`.
