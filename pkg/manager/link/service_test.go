package link

import (
	"io"
	"net/http"
	"testing"

	"github.com/sirrobot01/decypharr/pkg/debrid/types"
)

func TestValidateLinkUsesProviderClient(t *testing.T) {
	var providerCalled bool
	providerClient := &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		providerCalled = true
		return okResponse(r), nil
	})}
	defaultCalled := false
	service := &Service{
		httpClient: &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
			defaultCalled = true
			return okResponse(r), nil
		})},
		clientFor: func(provider string) *http.Client {
			if provider == "AllDebrid" {
				return providerClient
			}
			return nil
		},
	}

	err := service.validateLink(t.Context(), &types.DownloadLink{
		Debrid:       "AllDebrid",
		Filename:     "file.mkv",
		DownloadLink: "https://cdn.example/file.mkv",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !providerCalled || defaultCalled {
		t.Fatalf("provider client called = %v, default client called = %v", providerCalled, defaultCalled)
	}
}

func TestValidateLinkFallsBackToDefaultClient(t *testing.T) {
	defaultCalled := false
	service := &Service{
		httpClient: &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
			defaultCalled = true
			return okResponse(r), nil
		})},
		clientFor: func(string) *http.Client { return nil },
	}

	err := service.validateLink(t.Context(), &types.DownloadLink{
		Debrid:       "RealDebrid",
		Filename:     "file.mkv",
		DownloadLink: "https://cdn.example/file.mkv",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !defaultCalled {
		t.Fatal("default client was not used")
	}
}

func okResponse(r *http.Request) *http.Response {
	return &http.Response{
		StatusCode: http.StatusOK,
		Header:     make(http.Header),
		Body:       io.NopCloser(http.NoBody),
		Request:    r,
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return f(r)
}
