# Medical Scheduling Platform

This project implements a three-service medical scheduling platform in Go using Clean Architecture and microservices patterns.

The system is split into:

- **doctor-service**: owns doctor profile data and publishes events via NATS.
- **appointment-service**: owns appointment data and validates doctor existence through Doctor Service over gRPC. Also publishes events via NATS.
- **notification-service**: subscribes to events from other services via NATS and processes notifications.

## Project Overview

The platform demonstrates:

- separation of concerns inside each service;
- bounded contexts with separate data ownership;
- synchronous gRPC communication between services;
- asynchronous event-driven communication via NATS;
- basic failure handling when services depend on each other over the network.

Each service keeps business rules in the use case layer, persistence in the repository layer, and transport-specific logic in thin gRPC handlers.

## Architecture

```mermaid
graph TB
    subgraph "Clients"
        C[Client: Postman or grpcurl]
    end
    
    subgraph "Synchronous APIs"
        DS["Doctor Service<br/>(gRPC :9091)<br/>CreateDoctor, GetDoctor, ListDoctors"]
        AS["Appointment Service<br/>(gRPC :9092)<br/>CreateAppointment, GetAppointment, etc."]
    end
    
    subgraph "Data Layer"
        DDB[(Doctor DB<br/>owned by doctor-service)]
        ADB[(Appointment DB<br/>owned by appointment-service)]
    end
    
    subgraph "Asynchronous Messaging"
        NATS["NATS Message Broker<br/>(localhost:4222)"]
    end
    
    subgraph "Event Processing"
        NS["Notification Service<br/>Subscribes to events<br/>and processes notifications"]
    end
    
    C -->|gRPC :9091| DS
    C -->|gRPC :9092| AS
    DS --> DDB
    AS --> ADB
    AS -->|GetDoctor RPC| DS
    
    DS -->|Publish: doctors.created| NATS
    AS -->|Publish: appointments.created<br/>appointments.status_updated| NATS
    NATS -->|Subscribe & Process| NS
```

## Service Responsibilities

### Doctor Service RPCs

- CreateDoctor
- GetDoctor
- ListDoctors

Rules:

- full_name is required.
- email is required.
- email must be unique.

Events Published:

- `doctors.created`: emitted when a new doctor is created.

### Appointment Service RPCs

- CreateAppointment
- GetAppointment
- ListAppointments
- UpdateAppointmentStatus

Rules:

- title is required.
- doctor_id is required.
- the doctor must exist in Doctor Service;
- status must be new, in_progress, or done;
- transition from done back to new is rejected.

Events Published:

- `appointments.created`: emitted when a new appointment is created.
- `appointments.status_updated`: emitted when appointment status changes.

### Notification Service

Subscribes to events from Doctor Service and Appointment Service via NATS and processes notifications.

Subscribed Events:

- `doctors.created`: notifies about new doctors.
- `appointments.created`: notifies about new appointments.
- `appointments.status_updated`: notifies about appointment status changes.

## Folder Structure And Dependency Flow

Each service follows the same shape:

```text
service/
├── cmd/service-name/main.go
└── internal/
    ├── app/             # application wiring
    ├── logger/          # logging interface (notification-service specific)
    ├── model/           # domain entities
    ├── repository/      # persistence implementation (doctor/appointment services)
    ├── subscriber/      # event subscriber implementation (notification-service specific)
    ├── transport/       # gRPC handlers
    └── usecase/         # business logic and interfaces
```

Dependency direction points inward:

- handlers depend on use cases;
- use cases depend on interfaces;
- repositories implement repository interfaces;
- outbound gRPC clients implement use case interfaces;
- domain models do not depend on transport concerns.

## Inter-Service Communication

### Synchronous (gRPC)

Appointment Service calls Doctor Service over gRPC with GetDoctor before:

- creating an appointment;
- updating appointment status.

Appointment Service never accesses Doctor Service storage directly. This explicit RPC boundary keeps services decoupled at the data layer.

### Asynchronous (NATS)

- Doctor Service publishes `doctors.created` events when new doctors are created.
- Appointment Service publishes `appointments.created` and `appointments.status_updated` events for appointment state changes.
- Notification Service subscribes to these events and processes them asynchronously without blocking the originating services.

This decouples notification logic from core business operations.

## Event Contract

Events are published as JSON to NATS. Notification Service logs a JSON line for each event with fields `time`, `subject`, and `event`.

Example payloads:

```json
{
  "event_type": "doctors.created",
  "occurred_at": "2026-05-02T10:15:30Z",
  "id": "2a4b6d2c-5b9d-4c75-9a7c-2d6f8f2c6f32",
  "full_name": "Dr. Aisha Seitkali",
  "specialization": "Cardiology",
  "email": "a.seitkali@clinic.kz"
}
```

```json
{
  "event_type": "appointments.status_updated",
  "occurred_at": "2026-05-02T10:25:30Z",
  "id": "c1d2c3e4-1a2b-3c4d-5e6f-7a8b9c0d1e2f",
  "old_status": "new",
  "new_status": "in_progress"
}
```

Notification log example:

```json
{"time":"2026-05-02T10:25:30Z","subject":"appointments.status_updated","event":{"event_type":"appointments.status_updated","occurred_at":"2026-05-02T10:25:30Z","id":"c1d2c3e4-1a2b-3c4d-5e6f-7a8b9c0d1e2f","old_status":"new","new_status":"in_progress"}}
```

## Broker Choice

NATS is lightweight and low-latency, which fits fire-and-forget event delivery between internal services. RabbitMQ would add durable queues, routing flexibility, and acknowledgements, but at higher operational and protocol complexity. For this assignment the minimal broker setup and gRPC-first services made NATS the pragmatic choice.

## Failure Scenario

### gRPC (Synchronous)

If Doctor Service is unavailable when Appointment Service tries to create or update an appointment:

- the operation is rejected;
- Appointment Service logs verification failure internally;
- gRPC returns Unavailable with a descriptive message.

Current resilience is intentionally basic for the assignment:

- a 2-second timeout is configured on outbound gRPC client;
- no retry policy is applied;
- no circuit breaker is implemented.

### NATS (Asynchronous)

If NATS is unavailable when Doctor Service or Appointment Service tries to publish an event:

- the service gracefully handles the error;
- the core operation (create doctor/appointment) still succeeds;
- events may be lost if publishing fails (no persistent queue).

If Notification Service loses connection to NATS:

- it attempts to reconnect with exponential backoff (max 10 retries, up to 32 seconds);
- events published during disconnection are not replayed (fire-and-forget model);
- the service logs connection failures for debugging.

## Consistency Trade-offs and Design Decisions

### Fire-and-Forget Event Delivery (Current Implementation)

This implementation prioritizes **availability and performance** over guaranteed delivery:

- **No persistent queue**: Events published to NATS are not persisted to disk. If the broker crashes, in-flight events are lost.
- **No acknowledgements**: Services do not wait for Notification Service to confirm event processing before returning success to clients.
- **Best-effort delivery**: The core business operation (create doctor, update appointment) succeeds regardless of whether event publishing succeeds.

**When this is acceptable:**
- Notifications are informational and loss is tolerable (e.g., audit logging, metrics).
- The platform prioritizes low latency and high availability.

### Alternative Approaches (Not Implemented)

#### 1. **Outbox Pattern**
- Store events in a local transaction alongside the entity change (e.g., in `doctors_events` table).
- A separate **Outbox Poller** process reads and publishes events to the broker.
- Guarantees: Events are never lost (persisted locally) and exactly-once delivery is easier to achieve.
- Trade-off: Adds complexity (poller process, cleanup jobs), higher database load.

#### 2. **NATS JetStream**
- Enable NATS JetStream for persistent, replicated event streams.
- Services publish to JetStream subjects instead of core NATS subjects.
- Notification Service consumes via durable consumer groups with acknowledgement.
- Guarantees: Events are persisted and replayed on re-subscription.
- Trade-off: Increases broker resource requirements and protocol complexity.

#### 3. **RabbitMQ with Durable Queues**
- Use RabbitMQ exchanges and queues with `durable=true` and `auto-ack=false`.
- Explicit acknowledgements ensure messages are not lost.
- Guarantees: Messages persist to disk; no loss on broker restart.
- Trade-off: Heavier operational footprint; higher latency compared to NATS.

### Broker Comparison: NATS vs RabbitMQ

| Aspect | NATS | RabbitMQ |
|--------|------|----------|
| **Protocol** | Simple binary (fast) | AMQP (feature-rich) |
| **Message Persistence** | Optional (JetStream) | Built-in (queues) |
| **Delivery Guarantee** | At-most-once (core) | At-least-once (explicit acks) |
| **Latency** | Sub-millisecond | Low but higher than NATS |
| **Resource Usage** | Minimal (single binary) | Moderate (Erlang VM) |
| **Learning Curve** | Simple pub/sub model | Complex routing and bindings |
| **Operational Overhead** | Very low | Moderate (queue management) |
| **Production Readiness** | High (for fire-and-forget) | High (for guaranteed delivery) |

For this assignment, NATS was chosen because:
- Simple pub/sub model aligns with microservice event broadcasting.
- Low operational overhead for a teaching project.
- Fire-and-forget semantics are appropriate for notification use cases.

## Database Migrations

Migrations are automatically applied during service startup (before gRPC server initialization). The system uses `golang-migrate` with file-based migrations stored in `migrations/` folders.

### Automatic Migration on Startup

When you run `docker compose up -d` or start services locally:

1. Each service (doctor, appointment) connects to its database.
2. Migrations from the `migrations/` folder are discovered and executed in order.
3. Only new (unapplied) migrations run; previously applied migrations are skipped.
4. If migration fails, the service exits with an error.

Migration files follow the naming convention:
```
{VERSION}_{DESCRIPTION}.up.sql     (apply migration)
{VERSION}_{DESCRIPTION}.down.sql   (revert migration)
```

### Manual Migration Management

If you need to manually apply or revert migrations outside of service startup:

#### Using golang-migrate CLI

Install golang-migrate:

```bash
# macOS
brew install golang-migrate

# Linux
curl -L https://github.com/golang-migrate/migrate/releases/download/v4.16.2/migrate.linux-amd64.tar.gz | tar xvz
sudo mv migrate /usr/local/bin/

# Windows
choco install migrate
```

#### Apply all pending migrations

```bash
# For doctor service
migrate -path doctor-service/migrations \
  -database "postgres://postgres:postgres@localhost:5432/doctors?sslmode=disable" up

# For appointment service
migrate -path appointment-service/migrations \
  -database "postgres://postgres:postgres@localhost:5432/appointments?sslmode=disable" up
```

#### Revert the last N migrations

```bash
# Revert 1 migration
migrate -path doctor-service/migrations \
  -database "postgres://postgres:postgres@localhost:5432/doctors?sslmode=disable" down 1

# Revert all migrations
migrate -path doctor-service/migrations \
  -database "postgres://postgres:postgres@localhost:5432/doctors?sslmode=disable" down
```

#### Check migration status

```bash
migrate -path doctor-service/migrations \
  -database "postgres://postgres:postgres@localhost:5432/doctors?sslmode=disable" version
```

### Migration Files

**Doctor Service:**
- `000001_create_doctors.up.sql` — creates the `doctors` table with UUID primary key, unique email constraint.
- `000001_create_doctors.down.sql` — drops the `doctors` table.

**Appointment Service:**
- `000001_create_appointments.up.sql` — creates the `appointments` table with UUID primary key, foreign key reference to doctors.
- `000001_create_appointments.down.sql` — drops the `appointments` table.

## Prerequisites

- Go 1.25+
- protoc compiler
- protoc-gen-go plugin
- protoc-gen-go-grpc plugin
- Docker and Docker Compose (for running the full stack with PostgreSQL and NATS)

## Environment Variables

| Service | Variable | Default | Description |
| --- | --- | --- | --- |
| doctor-service | DATABASE_URL | postgres://postgres:postgres@db:5432/doctors?sslmode=disable | Postgres connection string |
| doctor-service | NATS_URL | nats://nats:4222 | NATS broker URL |
| doctor-service | GRPC_PORT | 9091 | gRPC listen port |
| appointment-service | DATABASE_URL | postgres://postgres:postgres@db:5432/appointments?sslmode=disable | Postgres connection string |
| appointment-service | NATS_URL | nats://nats:4222 | NATS broker URL |
| appointment-service | GRPC_PORT | 9092 | gRPC listen port |
| appointment-service | DOCTOR_SERVICE_ADDR | doctor-service:9091 | Doctor Service gRPC address |
| notification-service | NATS_URL | nats://nats:4222 | NATS broker URL |

## Docker Compose (PostgreSQL + NATS + Services)

From repository root:

```bash
docker compose up -d --build
```

Check status:

```bash
docker compose ps
```

Follow logs:

```bash
docker compose logs -f doctor-service
docker compose logs -f appointment-service
docker compose logs -f notification-service
```

### Install protoc and plugins

Windows (example with winget):

```powershell
winget install ProtocolBuffers.Protobuf
go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
```

Linux/macOS:

```bash
# install protoc with your package manager first
go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
```

Ensure Go bin is in PATH so protoc can find plugins.

## Regenerate gRPC Stubs

From repository root:

```bash
make proto-doctor
make proto-appointment
```

Or run protoc commands directly from each service folder as defined in Makefile.

## How To Run (Local)

### 1. Install service dependencies

From repository root:

```bash
cd doctor-service && go mod tidy
cd ../appointment-service && go mod tidy
cd ../notification-service && go mod tidy
```

### 2. Start PostgreSQL and NATS

If you are not using Docker Compose, make sure PostgreSQL and NATS are running and accessible via the env vars above.

### 3. Start Doctor Service (gRPC)

```bash
cd doctor-service
DATABASE_URL=postgres://postgres:postgres@localhost:5432/doctors?sslmode=disable \
NATS_URL=nats://localhost:4222 \
GRPC_PORT=9091 \
go run ./cmd/doctor-service
```

### 4. Start Appointment Service (gRPC)

In another terminal:

```bash
cd appointment-service
DATABASE_URL=postgres://postgres:postgres@localhost:5432/appointments?sslmode=disable \
NATS_URL=nats://localhost:4222 \
GRPC_PORT=9092 \
DOCTOR_SERVICE_ADDR=localhost:9091 \
go run ./cmd/appointment-service
```

### 5. Start Notification Service

In another terminal:

```bash
cd notification-service
NATS_URL=nats://localhost:4222 \
go run ./cmd/notification
```

Migrations are executed automatically on service startup via golang-migrate.

## gRPC Testing Artifact (grpcurl)

These commands demonstrate all RPCs required by the assignment.

### Doctor Service

CreateDoctor:

```bash
grpcurl -plaintext \
  -import-path doctor-service/proto \
  -proto doctor.proto \
  -d '{"full_name":"Dr. Aisha Seitkali","specialization":"Cardiology","email":"a.seitkali@clinic.kz"}' \
  localhost:9091 doctor.DoctorService/CreateDoctor
```

GetDoctor:

```bash
grpcurl -plaintext \
  -import-path doctor-service/proto \
  -proto doctor.proto \
  -d '{"id":"doctor-1"}' \
  localhost:9091 doctor.DoctorService/GetDoctor
```

ListDoctors:

```bash
grpcurl -plaintext \
  -import-path doctor-service/proto \
  -proto doctor.proto \
  -d '{}' \
  localhost:9091 doctor.DoctorService/ListDoctors
```

### Appointment Service

CreateAppointment:

```bash
grpcurl -plaintext \
  -import-path appointment-service/proto \
  -proto appointment.proto \
  -d '{"title":"Initial cardiac consultation","description":"Patient referred for palpitations","doctor_id":"doctor-1"}' \
  localhost:9092 appointment.AppointmentService/CreateAppointment
```

GetAppointment:

```bash
grpcurl -plaintext \
  -import-path appointment-service/proto \
  -proto appointment.proto \
  -d '{"id":"appointment-1"}' \
  localhost:9092 appointment.AppointmentService/GetAppointment
```

ListAppointments:

```bash
grpcurl -plaintext \
  -import-path appointment-service/proto \
  -proto appointment.proto \
  -d '{}' \
  localhost:9092 appointment.AppointmentService/ListAppointments
```

UpdateAppointmentStatus:

```bash
grpcurl -plaintext \
  -import-path appointment-service/proto \
  -proto appointment.proto \
  -d '{"id":"appointment-1","status":"in_progress"}' \
  localhost:9092 appointment.AppointmentService/UpdateAppointmentStatus
```

### Expected Notification Output

When the Notification Service processes an event, it logs a single JSON line like:

```json
{"time":"2026-05-02T10:25:30Z","subject":"appointments.status_updated","event":{"event_type":"appointments.status_updated","occurred_at":"2026-05-02T10:25:30Z","id":"c1d2c3e4-1a2b-3c4d-5e6f-7a8b9c0d1e2f","old_status":"new","new_status":"in_progress"}}
```

## Notes

- Doctor and Appointment services use PostgreSQL via pgx/v5.
- Migrations are applied automatically on startup using golang-migrate.
