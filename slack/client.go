package slack

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/Invicton-Labs/go-common/aws/ssm"
	"github.com/Invicton-Labs/go-common/log"
	"github.com/Invicton-Labs/go-stackerr"
	"github.com/slack-go/slack"
)

type SlackParameter struct {
	Token                  string `json:"token"`
	StatusMessageChannel   string `json:"status_message_channel"`
	StatusMessageTimestamp string `json:"status_message_timestamp"`
	MonitoringChannel      string `json:"monitoring_channel"`
}

type slackLogger struct {
	ddl log.DynamicDefaultLogger
}

type Client struct {
	*slack.Client
	parameters SlackParameter
}

func (c *Client) UpdateStatusMessage(blocks ...slack.Block) stackerr.Error {
	now := time.Now()
	blocks = append([]slack.Block{
		slack.NewHeaderBlock(slack.NewTextBlockObject(slack.PlainTextType, "Status Tracker", false, false)),
		slack.NewContextBlock("", slack.NewTextBlockObject(slack.MarkdownType, fmt.Sprintf("*Last Updated:* <!date^%d^{date_num} {time_secs}|%s>", now.Unix(), now.UTC().Format(time.RFC3339)), false, false)),
	}, blocks...)
	if _, _, _, err := c.UpdateMessage(c.parameters.StatusMessageChannel, c.parameters.StatusMessageTimestamp, slack.MsgOptionBlocks(
		blocks...,
	)); err != nil {
		return stackerr.Wrap(err)
	}
	return nil
}

func (sl *slackLogger) Output(calldepth int, message string) error {
	sl.ddl.Logger().WithAdditionalSkippedFrames(calldepth + 1).Infof(message)
	return nil
}

func GetParameter(ctx context.Context, ssmParamName string) (*SlackParameter, stackerr.Error) {
	// Load the secret from Secrets Manager
	slackParamString, err := ssm.GetSsmParameter(ctx, ssmParamName)
	if err != nil {
		return nil, err
	}
	parameter := SlackParameter{}
	if err := json.Unmarshal([]byte(*slackParamString), &parameter); err != nil {
		return nil, stackerr.Wrap(err)
	}
	if parameter.Token == "" {
		return nil, stackerr.Errorf("No 'token' found in Slack SSM parameter")
	}
	if parameter.MonitoringChannel == "" {
		return nil, stackerr.Errorf("No 'monitoring_channel' found in Slack SSM parameter")
	}
	if parameter.StatusMessageTimestamp == "" {
		return nil, stackerr.Errorf("No 'status_message_channel' found in Slack SSM parameter")
	}
	if parameter.StatusMessageTimestamp == "" {
		return nil, stackerr.Errorf("No 'status_message_timestamp' found in Slack SSM parameter")
	}
	return &parameter, nil
}

func NewClient(params *SlackParameter, httpClient *http.Client) *Client {
	return &Client{
		Client: slack.New(params.Token, slack.OptionDebug(false), slack.OptionHTTPClient(httpClient), slack.OptionLog(&slackLogger{
			ddl: log.NewDynamicDefaultLogger(func(input log.NewInput) log.NewInput {
				// Remove any write hooks for this logger, since that could create a recursive loop (slack error going to slack)
				input.WriteHooks = nil
				return input
			}),
		})),
		parameters: *params,
	}
}
