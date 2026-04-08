package client

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"strings"
	"time"

	"appointment-service/internal/usecase"
)

type DoctorHTTPClient struct {
	baseURL string
	client  *http.Client
}

func NewDoctorHTTPClient(baseURL string, timeout time.Duration) *DoctorHTTPClient {
	return &DoctorHTTPClient{
		baseURL: strings.TrimRight(baseURL, "/"),
		client: &http.Client{
			Timeout: timeout,
		},
	}
}

func (c *DoctorHTTPClient) VerifyDoctorExists(ctx context.Context, doctorID string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fmt.Sprintf("%s/doctors/%s", c.baseURL, doctorID), nil)
	if err != nil {
		return err
	}

	resp, err := c.client.Do(req)
	if err != nil {
		if isNetworkError(err) {
			return usecase.ErrDoctorServiceUnavailable
		}
		return err
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusOK:
		return nil
	case http.StatusNotFound:
		return usecase.ErrDoctorNotFound
	default:
		return usecase.ErrDoctorServiceUnavailable
	}
}

func isNetworkError(err error) bool {
	var netErr net.Error
	return errors.As(err, &netErr) || errors.Is(err, context.DeadlineExceeded)
}
