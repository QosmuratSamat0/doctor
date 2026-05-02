test:
	cd doctor-service && go test ./...
	cd appointment-service && go test ./...

run-doctor:
	cd doctor-service && go run ./cmd/doctor-service

run-appointment:
	cd appointment-service && go run ./cmd/appointment-service

proto-doctor:
	cd doctor-service && protoc \
		--proto_path=proto \
		--go_out=proto --go_opt=paths=source_relative \
		--go-grpc_out=proto --go-grpc_opt=paths=source_relative \
		proto/doctor.proto

proto-appointment:
	cd appointment-service && protoc \
		--proto_path=proto \
		--go_out=proto --go_opt=paths=source_relative \
		--go-grpc_out=proto --go-grpc_opt=paths=source_relative \
		proto/appointment.proto

MIGRATE_IMAGE=migrate/migrate

migrate-doctor-up:
	docker run -v $(PWD)/doctor-service/migrations:/migrations --network host $(MIGRATE_IMAGE) -path=/migrations/ -database "postgres://postgres:postgres@localhost:5432/doctors?sslmode=disable" up

migrate-doctor-down:
	docker run -v $(PWD)/doctor-service/migrations:/migrations --network host $(MIGRATE_IMAGE) -path=/migrations/ -database "postgres://postgres:postgres@localhost:5432/doctors?sslmode=disable" down

migrate-appointment-up:
	docker run -v $(PWD)/appointment-service/migrations:/migrations --network host $(MIGRATE_IMAGE) -path=/migrations/ -database "postgres://postgres:postgres@localhost:5432/appointments?sslmode=disable" up

migrate-appointment-down:
	docker run -v $(PWD)/appointment-service/migrations:/migrations --network host $(MIGRATE_IMAGE) -path=/migrations/ -database "postgres://postgres:postgres@localhost:5432/appointments?sslmode=disable" down

up:
	docker compose up -d

down:
	docker compose down
