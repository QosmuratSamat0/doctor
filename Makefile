test:
	cd doctor-service && go test ./...
	cd appointment-service && go test ./...

run-doctor:
	cd doctor-service && go run ./cmd/doctor-service

run-appointment:
	cd appointment-service && DOCTOR_SERVICE_URL=http://localhost:8081 go run ./cmd/appointment-service
