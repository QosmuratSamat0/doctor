package client

import (
	"context"
	"errors"
	"time"

	"appointment-service/internal/usecase"
	doctorpb "doctor-service/proto"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
)

type DoctorGRPCClient struct {
	conn           *grpc.ClientConn
	client         doctorpb.DoctorServiceClient
	requestTimeout time.Duration
}

func NewDoctorGRPCClient(address string, requestTimeout time.Duration) (*DoctorGRPCClient, error) {
	conn, err := grpc.Dial(address, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, err
	}

	return &DoctorGRPCClient{
		conn:           conn,
		client:         doctorpb.NewDoctorServiceClient(conn),
		requestTimeout: requestTimeout,
	}, nil
}

func (c *DoctorGRPCClient) Close() error {
	return c.conn.Close()
}

func (c *DoctorGRPCClient) VerifyDoctorExists(ctx context.Context, doctorID string) error {
	requestCtx := ctx
	var cancel context.CancelFunc
	if c.requestTimeout > 0 {
		requestCtx, cancel = context.WithTimeout(ctx, c.requestTimeout)
		defer cancel()
	}

	_, err := c.client.GetDoctor(requestCtx, &doctorpb.GetDoctorRequest{Id: doctorID})
	if err == nil {
		return nil
	}

	if code := status.Code(err); code == codes.NotFound {
		return usecase.ErrDoctorNotFound
	} else if code == codes.Unavailable || code == codes.DeadlineExceeded {
		return usecase.ErrDoctorServiceUnavailable
	}

	if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
		return usecase.ErrDoctorServiceUnavailable
	}

	return err
}
