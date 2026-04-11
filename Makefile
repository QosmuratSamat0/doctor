test:
	cd doctor-service && go test ./...
	cd appointment-service && go test ./...

run-doctor:
	cd doctor-service && go run ./cmd/doctor-service

run-appointment:
	cd appointment-service && DOCTOR_SERVICE_ADDR=localhost:9091 go run ./cmd/appointment-service

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
