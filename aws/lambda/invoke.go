package lambda

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/Invicton-Labs/go-common/aws/credentials"
	"github.com/Invicton-Labs/go-common/conversions"
	"github.com/Invicton-Labs/go-common/gensync"
	"github.com/Invicton-Labs/go-stackerr"
	awsarn "github.com/aws/aws-sdk-go-v2/aws/arn"
	"github.com/aws/aws-sdk-go-v2/service/lambda"
	"github.com/aws/aws-sdk-go-v2/service/lambda/types"
)

var lambdaClients gensync.Map[string, *lambda.Client]
var lambdaClientInitOnces gensync.Map[string, gensync.Once]

func getLambdaClient(ctx context.Context, region string) (*lambda.Client, stackerr.Error) {
	once, _ := lambdaClientInitOnces.LoadOrStore(region, gensync.Once{})
	if err := once.Do(func() stackerr.Error {
		creds, err := credentials.GetCredentialsProvider(ctx)
		if err != nil {
			return err
		}
		lambdaClient := lambda.New(lambda.Options{
			Region:      region,
			Credentials: creds,
		})
		lambdaClients.Store(region, lambdaClient)
		return nil
	}); err != nil {
		return nil, err
	}

	client, _ := lambdaClients.Load(region)
	return client, nil
}

func UpdateLambdaConfig(ctx context.Context, arn string, config lambda.UpdateFunctionConfigurationInput) stackerr.Error {
	parsedArn, cerr := awsarn.Parse(arn)
	if cerr != nil {
		return stackerr.Wrap(cerr)
	}
	client, err := getLambdaClient(ctx, parsedArn.Region)
	if err != nil {
		return err
	}
	config.FunctionName = conversions.GetPtr(arn)
	resp, cerr := client.UpdateFunctionConfiguration(ctx, &config)
	if cerr != nil {
		return stackerr.Wrap(cerr)
	}
	lastStatus := resp.LastUpdateStatus
	lastStatusReason := resp.LastUpdateStatusReason
	if lastStatus == types.LastUpdateStatusInProgress {
		for {
			select {
			case <-ctx.Done():
				return stackerr.Wrap(ctx.Err())
			case <-time.After(2 * time.Second):
			}

			cfg, cerr := client.GetFunctionConfiguration(ctx, &lambda.GetFunctionConfigurationInput{
				FunctionName: conversions.GetPtr(arn),
			})
			if cerr != nil {
				return stackerr.Wrap(cerr)
			}
			if cfg.LastUpdateStatus != types.LastUpdateStatusInProgress {
				lastStatus = cfg.LastUpdateStatus
				lastStatusReason = cfg.LastUpdateStatusReason
				break
			}
		}
	}

	if lastStatus != types.LastUpdateStatusSuccessful {
		reason := "Unknown"
		if lastStatusReason != nil {
			reason = *lastStatusReason
		}
		return stackerr.Errorf("Failed to update Lambda function: %s", reason).With(map[string]any{
			"arn": arn,
		})
	}
	return nil
}

func ForceLambdaReset(ctx context.Context, arn string) stackerr.Error {
	parsedArn, cerr := awsarn.Parse(arn)
	if cerr != nil {
		return stackerr.Wrap(cerr)
	}
	client, err := getLambdaClient(ctx, parsedArn.Region)
	if err != nil {
		return err
	}
	cfg, cerr := client.GetFunctionConfiguration(ctx, &lambda.GetFunctionConfigurationInput{
		FunctionName: conversions.GetPtr(arn),
	})
	if cerr != nil {
		return stackerr.Wrap(cerr)
	}
	originalMemorySize := *cfg.MemorySize
	newMemorySize := originalMemorySize + 1

	// Change the memory size
	if err := UpdateLambdaConfig(ctx, arn, lambda.UpdateFunctionConfigurationInput{
		MemorySize: &newMemorySize,
	}); err != nil {
		return err
	}

	// And change it back
	if err := UpdateLambdaConfig(ctx, arn, lambda.UpdateFunctionConfigurationInput{
		MemorySize: &originalMemorySize,
	}); err != nil {
		return err
	}
	return nil
}

func Invoke(ctx context.Context, arn string, payload []byte) (responsePayload []byte, err stackerr.Error) {
	parsedArn, cerr := awsarn.Parse(arn)
	if cerr != nil {
		return nil, stackerr.Wrap(cerr)
	}
	client, err := getLambdaClient(ctx, parsedArn.Region)
	if err != nil {
		return nil, err
	}
	invokeOutput, cerr := client.Invoke(ctx, &lambda.InvokeInput{
		FunctionName:   conversions.GetPtr(arn),
		InvocationType: types.InvocationTypeRequestResponse,
		Payload:        payload,
	})
	if cerr != nil {
		return nil, stackerr.Wrap(cerr)
	}
	if invokeOutput.FunctionError != nil || invokeOutput.StatusCode != 200 {
		fields := map[string]any{
			"arn":              arn,
			"status_code":      invokeOutput.StatusCode,
			"invoked_logs_url": LogGroupUrl(parsedArn.Region, fmt.Sprintf("/aws/lambda/%s", strings.SplitN(parsedArn.Resource, ":", 2)[1])),
		}
		if invokeOutput.FunctionError != nil {
			fields["function_err"] = *invokeOutput.FunctionError
			if invokeOutput.Payload != nil {
				type errPayload struct {
					ErrMsg  string `json:"errorMessage"`
					ErrType string `json:"errorType"`
				}
				p := errPayload{}
				if err := json.Unmarshal(invokeOutput.Payload, &p); err == nil {
					fields["err_msg"] = p.ErrMsg
					fields["err_type"] = p.ErrType
				}
			}
			return nil, stackerr.Errorf("Lambda invocation failed").With(fields)
		} else {
			return nil, stackerr.Errorf("%d", invokeOutput.StatusCode).With(fields)
		}
	}
	return invokeOutput.Payload, nil
}
