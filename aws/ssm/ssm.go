package ssm

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/Invicton-Labs/go-common/conversions"
	"github.com/Invicton-Labs/go-stackerr"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
	"github.com/aws/aws-sdk-go/aws/arn"
)

var ssmClient *ssm.Client

func getSsmClient(ctx context.Context, region *string) *ssm.Client {
	// If a region is specified, create a client specifically
	// for that region
	if region != nil {
		cfg, err := config.LoadDefaultConfig(ctx)
		if err != nil {
			panic(err)
		}
		cfg.Region = *region
		return ssm.NewFromConfig(cfg)
	}

	if ssmClient != nil {
		return ssmClient
	}
	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		panic(err)
	}
	ssmClient = ssm.NewFromConfig(cfg)
	return ssmClient
}

func GetSsmParameter(ctx context.Context, parameter string) (*string, stackerr.Error) {
	name := parameter
	var region *string

	// Handle ARN versions of parameter names
	if arn.IsARN(parameter) {
		a, err := arn.Parse(parameter)
		if err != nil {
			return nil, stackerr.Wrap(err)
		}
		resourcePrefix := "parameter/"
		if !strings.HasPrefix(strings.ToLower(a.Resource), resourcePrefix) {
			return nil, stackerr.Errorf("SSM parameter ARN resource does not begin with 'parameter/': %s", parameter)
		}
		name = a.Resource[len(resourcePrefix)-1:]
		region = &a.Region
	}

	client := getSsmClient(ctx, region)
	param, err := client.GetParameter(ctx, &ssm.GetParameterInput{
		Name:           &name,
		WithDecryption: conversions.GetPtr(true),
	})
	if err != nil {
		return nil, stackerr.Wrap(err)
	}

	return param.Parameter.Value, nil
}

func GetSsmParameterUnmarshaled[T any](ctx context.Context, parameter string) (*T, stackerr.Error) {
	strval, err := GetSsmParameter(ctx, parameter)
	if err != nil {
		return nil, err
	}
	var t T
	if err := json.Unmarshal([]byte(*strval), &t); err != nil {
		return nil, stackerr.Wrap(err)
	}
	return &t, nil
}
