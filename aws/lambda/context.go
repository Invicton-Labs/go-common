package lambda

import (
	"context"

	"github.com/Invicton-Labs/go-stackerr"
	"github.com/aws/aws-lambda-go/lambdacontext"
	"github.com/aws/aws-sdk-go-v2/aws/arn"
)

type LambdaMeta struct {
	LambdaArn       string `json:"lambda_arn"`
	AccountId       string `json:"account_id"`
	Region          string `json:"region"`
	FunctionName    string `json:"function_name"`
	FunctionVersion string `json:"function_version"`
	LogGroupName    string `json:"log_group_name"`
	LogStreamName   string `json:"log_stream_name"`
	RequestId       string `json:"request_id"`
}

func MetaFromContext(ctx context.Context) (LambdaMeta, stackerr.Error) {
	lc, ok := lambdacontext.FromContext(ctx)
	if !ok {
		return LambdaMeta{}, stackerr.Errorf("Failed to load Lambda context from context")
	}
	a, err := arn.Parse(lc.InvokedFunctionArn)
	if err != nil {
		return LambdaMeta{}, stackerr.Wrap(err)
	}

	return LambdaMeta{
		LambdaArn:       lc.InvokedFunctionArn,
		AccountId:       a.AccountID,
		Region:          a.Region,
		FunctionName:    lambdacontext.FunctionName,
		FunctionVersion: lambdacontext.FunctionVersion,
		LogGroupName:    lambdacontext.LogGroupName,
		LogStreamName:   lambdacontext.LogStreamName,
		RequestId:       lc.AwsRequestID,
	}, nil
}
