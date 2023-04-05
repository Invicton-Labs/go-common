package retryablehttp

import (
	"bytes"
	"context"
	"errors"
	"io"
	"math"
	"net/http"
	"reflect"
	"strings"
	"time"

	"github.com/Invicton-Labs/go-common/log"
	"github.com/Invicton-Labs/go-stackerr"
	"github.com/die-net/lrucache"
	"github.com/gregjones/httpcache"
	hashicorphttp "github.com/hashicorp/go-retryablehttp"
	"golang.org/x/net/http2"
)

type NewClientInput struct {
	// The maximum size, in bytes, of the cache. A cache will
	// only be used if this value is non-zero.
	CacheMaxSizeBytes int64
	// 0 for never expiring
	CacheMaxAgeSeconds int64
	// The base transport settings to use.
	// This is not used for embedded Tor clients.
	RoundTripper http.RoundTripper
	// The maximum number of retries for each request. If less
	// than 0, it will be treated as unlimited (technically,
	// max int32)
	MaxRetries int
	// The minimum amount of time to wait between retries
	RetryWaitMin time.Duration
	// The maximum amount of time to wait between retries
	RetryWaitMax time.Duration
	// The logger to use. If not provided, the default one
	// will be used.
	Logger hashicorphttp.LeveledLogger
	// A custom backoff function, if desired
	Backoff func(min, max time.Duration, attemptNum int, resp *http.Response) time.Duration
	// A custom retry function, if desired
	CheckRetry func(ctx context.Context, resp *http.Response, httpErr error) (bool, error)
}

var goAwayErrorType reflect.Type = reflect.TypeOf(http2.GoAwayError{})
var goAwayErrorPtrType reflect.Type = reflect.TypeOf(&http2.GoAwayError{})

func NewRoundTripper(input *NewClientInput) http.RoundTripper {

	retryableClient := hashicorphttp.NewClient()
	retryableClient.HTTPClient.Transport = input.RoundTripper

	if input.Logger != nil {
		retryableClient.Logger = input.Logger
	} else {
		retryableClient.Logger = GetRetryhttpLeveledLogger(nil)
	}
	if input.MaxRetries != 0 {
		if input.MaxRetries < 0 {
			retryableClient.RetryMax = math.MaxInt32
		} else {
			retryableClient.RetryMax = input.MaxRetries
		}
	}
	if input.RetryWaitMin != 0 {
		retryableClient.RetryWaitMin = input.RetryWaitMin
	}
	if input.RetryWaitMax != 0 {
		retryableClient.RetryWaitMax = input.RetryWaitMax
	}

	// If a cache should be used, wrap the transport in a cacher
	if input.CacheMaxSizeBytes > 0 {
		// Create an in-memory cache
		lcache := lrucache.New(input.CacheMaxSizeBytes, input.CacheMaxAgeSeconds)

		// Create a cached http client for the CCP APIs.
		cacheTransport := httpcache.NewTransport(lcache)
		cacheTransport.Transport = retryableClient.HTTPClient.Transport
		// Set the client transport to be the wrapped cache transport
		retryableClient.HTTPClient.Transport = cacheTransport
	}

	// Use a custom backoff function that logs the error before calling the default backoff function
	retryableClient.Backoff = func(min, max time.Duration, attemptNum int, resp *http.Response) time.Duration {
		if resp == nil {
			log.Debugw("Failed HTTP request, cause unknown (response is nil)")
		} else {
			body, _ := GetAndRewindHttpResponseBody(resp)
			if body == nil {
				body = []byte{}
			}
			log.Debugw(
				"Failed HTTP request",
				"url", resp.Request.URL.String(),
				"status_code", resp.StatusCode,
				"status", resp.Status,
				"body", string(body),
				"attempt_number", attemptNum,
			)
		}
		// If a custom backoff function was specified, use it
		if input.Backoff != nil {
			return input.Backoff(min, max, attemptNum, resp)
		}
		// Otherwise, use the default
		return hashicorphttp.DefaultBackoff(min, max, attemptNum, resp)
	}

	// Wrap the retry policy to retry on 420 errors (error throttling)
	retryableClient.CheckRetry = func(ctx context.Context, resp *http.Response, httpErr error) (shouldRetry bool, err error) {
		if input.CheckRetry != nil {
			// If a custom retry function was specified, use it
			shouldRetry, err = input.CheckRetry(ctx, resp, httpErr)
			err = stackerr.Wrap(err)
		} else {
			// Otherwise, use the default
			shouldRetry, err = hashicorphttp.DefaultRetryPolicy(ctx, resp, httpErr)
			err = stackerr.Wrap(err)
		}

		// If there's no err describing the retry, but there was an HTTP error,
		// use the HTTP error to describe the retry.
		if err == nil && httpErr != nil {
			err = stackerr.Wrap(httpErr)
		}

		// If we haven't, so far, found any reason to retry, check some
		// special conditions.
		if !shouldRetry {

			// If there is no error, read the body to detect any
			// error that reading it might generate.
			// Specifically, this will detect GOAWAY errors from
			// the server that only appear during body reading.
			if err == nil {
				_, err = GetAndRewindHttpResponseBody(resp)
			}

			// If an error has been found, check it for specific error types
			if err != nil {
				unwrapped := err
				for unwrapped != nil {
					errType := reflect.TypeOf(unwrapped)
					if errType == goAwayErrorType ||
						errType == goAwayErrorPtrType ||
						strings.Contains(unwrapped.Error(), "http2: server sent GOAWAY") ||
						strings.Contains(unwrapped.Error(), "http2: client connection force closed") ||
						strings.Contains(unwrapped.Error(), "unexpected EOF") {
						shouldRetry = true
						break
					}
					unwrapped = errors.Unwrap(unwrapped)
				}
			}
		}

		// If we want to retry but no error has been specified, and there was an HTTP response,
		// use the HTTP response status to generate the error
		if shouldRetry && err == nil && resp != nil {
			err = stackerr.Errorf("%d: %s", resp.StatusCode, resp.Status)
		}

		return shouldRetry, err
	}
	return retryableClient.StandardClient().Transport
}

func GetAndRewindHttpResponseBody(resp *http.Response) ([]byte, stackerr.Error) {
	if resp == nil || resp.Body == nil {
		return nil, nil
	}
	b, err := io.ReadAll(resp.Body)
	resp.Body.Close()
	// Ensure that there's always a body, even if it's empty
	if b == nil {
		b = []byte{}
	}
	// Rewind the body. Always do this, even on an error,
	// as an error does not necessarily mean we don't need the body later.
	resp.Body = io.NopCloser(bytes.NewBuffer(b))
	if err != nil {
		return nil, stackerr.Wrap(err)
	}
	return b, nil
}
