package awscerts

import (
	"crypto/x509"
	"embed"
	"fmt"

	"github.com/Invicton-Labs/go-stackerr"
)

//go:embed bundles/*
var awsCertBundles embed.FS

func GetRootCertPool(region string) (*x509.CertPool, stackerr.Error) {
	pem, err := awsCertBundles.ReadFile(fmt.Sprintf("bundles/%s-bundle.pem", region))
	if err != nil {
		return nil, stackerr.Wrap(err)
	}
	rootCertPool := x509.NewCertPool()
	if ok := rootCertPool.AppendCertsFromPEM(pem); !ok {
		return nil, stackerr.Wrap(err)
	}
	return rootCertPool, nil
}
