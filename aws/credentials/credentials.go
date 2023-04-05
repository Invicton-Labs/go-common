package credentials

import (
	"context"

	"github.com/Invicton-Labs/go-common/gensync"
	"github.com/Invicton-Labs/go-stackerr"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
)

var commonCfg *aws.Config
var configOnce gensync.Once

func GetConfig(ctx context.Context) (*aws.Config, stackerr.Error) {
	if err := configOnce.Do(func() stackerr.Error {
		newCfg, err := config.LoadDefaultConfig(ctx)
		if err != nil {
			return stackerr.Wrap(err)
		}
		commonCfg = &newCfg
		return nil
	}); err != nil {
		return nil, err
	}
	return commonCfg, nil
}

func GetCredentialsProvider(ctx context.Context) (aws.CredentialsProvider, stackerr.Error) {
	cfg, err := GetConfig(ctx)
	if err != nil {
		return nil, err
	}
	return cfg.Credentials, nil
}

func GetCredentials(ctx context.Context) (aws.Credentials, stackerr.Error) {
	cfg, err := GetConfig(ctx)
	if err != nil {
		return aws.Credentials{}, err
	}
	creds, cerr := cfg.Credentials.Retrieve(ctx)
	if cerr != nil {
		return aws.Credentials{}, stackerr.Wrap(cerr)
	}
	return creds, nil
}
