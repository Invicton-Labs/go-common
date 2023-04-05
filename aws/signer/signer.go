package s3

import (
	"bytes"
	"context"
	"crypto/sha256"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/Invicton-Labs/go-common/aws/credentials"
	"github.com/Invicton-Labs/go-common/log"
	"github.com/Invicton-Labs/go-stackerr"
	v4 "github.com/aws/aws-sdk-go-v2/aws/signer/v4"
)

var signer *v4.Signer

func init() {
	signer = v4.NewSigner(func(signer *v4.SignerOptions) {
		signer.Logger = log.GetAwsLogger()
		signer.LogSigning = true
	})
}

func GetSigner() *v4.Signer {
	return signer
}

func getAndRewindHttpRequestBody(req *http.Request) ([]byte, stackerr.Error) {
	if req == nil || req.Body == nil {
		return nil, nil
	}
	b, err := io.ReadAll(req.Body)
	req.Body.Close()
	// Ensure that there's always a body, even if it's empty
	if b == nil {
		b = []byte{}
	}
	// Rewind the body. Always do this, even on an error,
	// as an error does not necessarily mean we don't need the body later.
	req.Body = io.NopCloser(bytes.NewBuffer(b))
	if err != nil {
		return nil, stackerr.Wrap(err)
	}
	return b, nil
}

// SignRequest will sign the given HTTP request (in-place modification) with the default AWS credentials
func SignRequest(ctx context.Context, req *http.Request) stackerr.Error {
	creds, err := credentials.GetCredentials(ctx)
	if err != nil {
		return err
	}

	body, err := getAndRewindHttpRequestBody(req)
	if err != nil {
		return err
	}

	bodyHash := sha256.Sum256(body)
	return stackerr.Wrap(signer.SignHTTP(ctx, creds, req, fmt.Sprintf("%x", bodyHash), "", "", time.Now()))
}
