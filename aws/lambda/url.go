package lambda

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	"github.com/Invicton-Labs/go-stackerr"
)

func LogGroupUrl(region string, group string) string {
	logGroup := strings.ReplaceAll(url.PathEscape(url.PathEscape(group)), "%", "$")
	logGroupUrl := fmt.Sprintf("https://%s.console.aws.amazon.com/cloudwatch/home?region=%s#logsV2:log-groups/log-group/%s/", region, region, logGroup)
	return logGroupUrl
}

func LogStreamUrl(region string, group string, stream string) string {
	logGroup := strings.ReplaceAll(url.PathEscape(url.PathEscape(group)), "%", "$")
	logStreamParam := url.QueryEscape(stream)
	logStreamUrl := fmt.Sprintf("https://%s.console.aws.amazon.com/cloudwatch/home?region=%s#logsV2:log-groups/log-group/%s/log-events/%s", region, region, logGroup, strings.ReplaceAll(url.PathEscape(logStreamParam), "%", "$"))
	return logStreamUrl
}

func FilteredLogStreamUrl(region string, group string, stream string, filter string) string {
	logGroup := strings.ReplaceAll(url.PathEscape(url.PathEscape(group)), "%", "$")
	logStreamParam := fmt.Sprintf("%s?filterPattern=%s", url.QueryEscape(stream), url.QueryEscape(filter))
	logStreamUrl := fmt.Sprintf("https://%s.console.aws.amazon.com/cloudwatch/home?region=%s#logsV2:log-groups/log-group/%s/log-events/%s", region, region, logGroup, strings.ReplaceAll(url.PathEscape(logStreamParam), "%", "$"))
	return logStreamUrl
}

func RequestIdLogStreamUrl(region string, group string, stream string, request_id string) string {
	return FilteredLogStreamUrl(region, group, stream, fmt.Sprintf("\"%s\"", request_id))
}

func RequestIdLogStreamUrlFromContext(ctx context.Context) (string, stackerr.Error) {
	meta, err := MetaFromContext(ctx)
	if err != nil {
		return "", err
	}
	return RequestIdLogStreamUrl(meta.Region, meta.LogGroupName, meta.LogStreamName, meta.RequestId), nil
}

func RequestIdLogStreamUrlFromMeta(meta LambdaMeta) string {
	return RequestIdLogStreamUrl(meta.Region, meta.LogGroupName, meta.LogStreamName, meta.RequestId)
}
