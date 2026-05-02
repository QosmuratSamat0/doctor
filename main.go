package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"syscall"
	"time"
)

type serviceProcess struct {
	name string
	cmd  *exec.Cmd
}

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	processes := []*serviceProcess{
		newServiceProcess(
			"doctor-service",
			"doctor-service",
			[]string{"go", "run", "./cmd/doctor-service"},
			nil,
		),
		newServiceProcess(
			"appointment-service",
			"appointment-service",
			[]string{"go", "run", "./cmd/appointment-service"},
			[]string{"DOCTOR_SERVICE_ADDR=localhost:9091"},
		),
	}

	for _, process := range processes {
		if err := process.cmd.Start(); err != nil {
			stopAll(processes)
			log.Fatalf("failed to start %s: %v", process.name, err)
		}
		log.Printf("%s started with PID %d", process.name, process.cmd.Process.Pid)
		time.Sleep(250 * time.Millisecond)
	}

	errCh := make(chan error, len(processes))
	for _, process := range processes {
		go func(process *serviceProcess) {
			errCh <- waitForProcess(process)
		}(process)
	}

	select {
	case <-ctx.Done():
		log.Println("shutdown signal received, stopping services")
		stopAll(processes)
		drainWaits(errCh, len(processes))
	case err := <-errCh:
		if err != nil {
			log.Printf("service exited with error: %v", err)
		}
		stopAll(processes)
		drainWaits(errCh, len(processes)-1)
		if err != nil {
			os.Exit(1)
		}
	}
}

func newServiceProcess(name, dir string, args []string, extraEnv []string) *serviceProcess {
	cmd := exec.Command(args[0], args[1:]...)
	cmd.Dir = dir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	cmd.Env = append(os.Environ(), extraEnv...)
	return &serviceProcess{name: name, cmd: cmd}
}

func waitForProcess(process *serviceProcess) error {
	if err := process.cmd.Wait(); err != nil {
		return fmt.Errorf("%s: %w", process.name, err)
	}
	return nil
}

func stopAll(processes []*serviceProcess) {
	for _, process := range processes {
		if process.cmd.Process == nil {
			continue
		}
		_ = process.cmd.Process.Signal(syscall.SIGTERM)
	}
}

func drainWaits(errCh <-chan error, remaining int) {
	for i := 0; i < remaining; i++ {
		<-errCh
	}
}
