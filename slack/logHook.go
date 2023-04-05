package slack

import (
	"context"
	"fmt"
	"math"
	"net/http"
	"strings"
	"time"

	"github.com/Invicton-Labs/go-common/collections"
	"github.com/Invicton-Labs/go-common/dateutils"
	"github.com/Invicton-Labs/go-common/log"
	"github.com/Invicton-Labs/go-common/numbers"
	retryablehttp "github.com/Invicton-Labs/go-common/retryable-http"
	"github.com/Invicton-Labs/go-common/slack/links"
	"github.com/Invicton-Labs/go-stackerr"
	"github.com/hashicorp/go-cleanhttp"
	"github.com/slack-go/slack"
	"go.uber.org/zap/zapcore"
)

func formatLevel(level zapcore.Level) string {
	switch level {
	case zapcore.DebugLevel:
		return "DEBUG"
	case zapcore.InfoLevel:
		return "INFO"
	case zapcore.WarnLevel:
		return "WARNING"
	case zapcore.ErrorLevel:
		return "ERROR"
	case zapcore.PanicLevel:
		return "PANIC"
	case zapcore.FatalLevel:
		return "FATAL"
	default:
		return fmt.Sprintf("UNKNOWN (%d)", level)
	}
}

func formatStackErr(err log.StackError) []slack.Block {
	blocks := make([]slack.Block, 0, 5)

	// Start with a divider
	blocks = append(blocks, slack.NewDividerBlock())

	var errHeader string
	if err.Key != "" {
		errHeader = fmt.Sprintf("*Error:* %s", err.Key)
	} else {
		errHeader = "*Error*"
	}
	blocks = append(blocks, slack.NewSectionBlock(slack.NewTextBlockObject(slack.MarkdownType, errHeader, false, false), nil, nil))

	fields := make([]*slack.TextBlockObject, 0, len(err.Fields)+1)
	if err.Key != "" {
		fields = append(fields, slack.NewTextBlockObject(slack.MarkdownType, fmt.Sprintf("*Key*\n%s", err.Key), false, false))
	}
	fields = append(fields, collections.TransformMapToSlice(err.Fields, func(key string, value any) *slack.TextBlockObject {
		return slack.NewTextBlockObject(slack.MarkdownType, fmt.Sprintf("*%s*\n%v", key, value), false, false)
	})...)

	// If there are fields, add them
	if len(fields) > 0 {
		blocks = append(blocks, slack.NewSectionBlock(nil, fields, nil))
	}

	// Add the error message
	blocks = append(blocks, slack.NewSectionBlock(slack.NewTextBlockObject(slack.PlainTextType, err.Message, false, false), nil, nil))

	// Add the stack traces
	blocks = append(blocks, slack.NewSectionBlock(slack.NewTextBlockObject(slack.MarkdownType, "```\n"+err.Stacktraces.Format()+"\n```", false, false), nil, nil))

	return blocks
}

// Formats a time as a
func formatTime(t time.Time) string {
	if t.IsZero() {
		return "Zero-time"
	}
	return fmt.Sprintf("<!date^%d^{date_num} {time_secs}|%s>", t.Unix(), t.Format(time.RFC3339))
}

func NewSlackHook(ctx context.Context, params *SlackParameter, level zapcore.Level) log.ZapWriteHook {

	httpClient := &http.Client{
		Transport: retryablehttp.NewRoundTripper(&retryablehttp.NewClientInput{
			Logger: retryablehttp.GetRetryhttpLeveledLogger(func(input log.NewInput) log.NewInput {
				// Ensure that the Slack logger doesn't try to use the Slack logger
				input.WriteHooks = nil
				return input
			}),
			RoundTripper: cleanhttp.DefaultPooledTransport(),
		}),
		Timeout: 5 * time.Second,
	}

	client := NewClient(params, httpClient)
	blockLengthLimit := 3000

	return func(e zapcore.Entry, fields map[string]zapcore.Field, errs []log.StackError, stacktraces stackerr.Stacks) stackerr.Error {
		if e.Level < level {
			return nil
		}
		payloadFields := []*slack.TextBlockObject{
			slack.NewTextBlockObject(slack.MarkdownType, fmt.Sprintf("*Time*\n%s", formatTime(e.Time)), false, false),
			slack.NewTextBlockObject(slack.MarkdownType, fmt.Sprintf("*Level*\n%s", formatLevel(e.Level)), false, false),
		}

		// Create a generator that goes through the fields in ascending order by key
		gen := collections.MapAscending(fields)
		for _, field, ok := gen(); ok; _, field, ok = gen() {
			var val string
			switch field.Type {
			case zapcore.BoolType:
				val = fmt.Sprintf("%t", field.Integer == 1)
				// ByteStringType indicates that the field carries UTF-8 encoded bytes.
			case zapcore.DurationType:
				val = field.Interface.(time.Duration).String()
				// Float64Type indicates that the field carries a float64.
			case zapcore.Float64Type:
				val = fmt.Sprintf("%f", math.Float64frombits(uint64(field.Integer)))
				// Float32Type indicates that the field carries a float32.
			case zapcore.Float32Type:
				val = fmt.Sprintf("%f", math.Float32frombits(uint32(field.Integer)))
				// Int64Type indicates that the field carries an int64.
			case zapcore.Int64Type:
				val = fmt.Sprintf("%d", field.Integer)
				// Int32Type indicates that the field carries an int32.
			case zapcore.Int32Type:
				val = fmt.Sprintf("%d", field.Integer)
				// Int16Type indicates that the field carries an int16.
			case zapcore.Int16Type:
				val = fmt.Sprintf("%d", field.Integer)
				// Int8Type indicates that the field carries an int8.
			case zapcore.Int8Type:
				val = fmt.Sprintf("%d", field.Integer)
				// StringType indicates that the field carries a string.
			case zapcore.StringType:
				val = field.String
				// TimeType indicates that the field carries a time.Time that is
				// representable by a UnixNano() stored as an int64.
			case zapcore.TimeType:
				val = formatTime(dateutils.TimeFromUnix(field.Integer))
				// TimeFullType indicates that the field carries a time.Time stored as-is.
			case zapcore.TimeFullType:
				val = formatTime(field.Interface.(time.Time))
				// Uint64Type indicates that the field carries a uint64.
			case zapcore.Uint64Type:
				val = fmt.Sprintf("%d", field.Integer)
				// Uint32Type indicates that the field carries a uint32.
			case zapcore.Uint32Type:
				val = fmt.Sprintf("%d", field.Integer)
				// Uint16Type indicates that the field carries a uint16.
			case zapcore.Uint16Type:
				val = fmt.Sprintf("%d", field.Integer)
				// Uint8Type indicates that the field carries a uint8.
			case zapcore.Uint8Type:
				val = fmt.Sprintf("%d", field.Integer)
				// UintptrType indicates that the field carries a uintptr.
			case zapcore.UintptrType:
				val = fmt.Sprint(field.Integer)
				// StringerType indicates that the field carries a fmt.Stringer.
			case zapcore.StringerType:
				val = field.Interface.(fmt.Stringer).String()
				// ErrorType indicates that the field carries an error.
			case zapcore.ErrorType:
				val = field.Interface.(error).Error()
			default:
				if field.String != "" {
					val = field.String
				} else if field.Interface != nil {
					switch v := field.Interface.(type) {
					case time.Time:
						val = formatTime(v)
					case *time.Time:
						val = formatTime(*v)
					case links.SlackLink:
						val = v.SlackFormat()
					default:
						val = fmt.Sprintf("%v", field.Interface)
					}
				} else if field.Integer != 0 {
					val = fmt.Sprint(field.Integer)
				} else {
					val = "N/A"
				}
			}
			msg := fmt.Sprintf("*%s*\n%s", field.Key, val)
			payloadFields = append(payloadFields, slack.NewTextBlockObject(slack.MarkdownType, msg[0:numbers.Min(len(msg), blockLengthLimit)], false, false))
		}
		blocks := []slack.Block{
			slack.NewHeaderBlock(slack.NewTextBlockObject(slack.PlainTextType, fmt.Sprintf(":rotating_light: Monitoring Alert: %s", e.LoggerName), false, false)),
			slack.NewSectionBlock(nil, payloadFields, nil),
		}
		if len(e.Message) > 0 {
			msg := "*Message*\n" + e.Message
			blocks = append(blocks, slack.NewSectionBlock(slack.NewTextBlockObject(slack.MarkdownType, msg[0:numbers.Min(len(msg), blockLengthLimit)], false, false), nil, nil))
		}

		// Add fields for each error
		for _, err := range errs {
			blocks = append(blocks, formatStackErr(err)...)
		}
		blocks = append(blocks, slack.NewDividerBlock())

		stackBlockLengthLimit := blockLengthLimit - 2*len("\n```\"")
		// Add the stack traces for the log itself
		if len(stacktraces) > 0 {
			blocks = append(blocks, slack.NewSectionBlock(slack.NewTextBlockObject(slack.MarkdownType, "*Log stacktrace:*", false, false), nil, nil))
			msgLines := strings.Split(stacktraces.Format(), "\n")
			msg := ""
			for _, l := range msgLines {
				// A single line can never have more than the block length
				if len(l) > stackBlockLengthLimit {
					l = l[0:stackBlockLengthLimit]
				}
				// Check if we would go over the limit if we append a newline and this line
				if len(msg)+1+len(l) > stackBlockLengthLimit {
					// If so, store the current message in a new block
					blocks = append(blocks, slack.NewSectionBlock(slack.NewTextBlockObject(slack.MarkdownType, "```"+msg+"```", false, false), nil, nil))
					// And set the next block to start with this line
					msg = l
				} else {
					// If there's any existing content, add a new line
					if len(msg) > 0 {
						msg += "\n"
					}
					// Append this line to the block message
					msg += l
				}
			}
			// If there's any message left, add it in a separate block
			if len(msg) > 0 {
				blocks = append(blocks, slack.NewSectionBlock(slack.NewTextBlockObject(slack.MarkdownType, "```"+msg+"```", false, false), nil, nil))
			}
			blocks = append(blocks,
				slack.NewDividerBlock(),
			)
		}

		if _, _, err := client.PostMessage(params.MonitoringChannel, slack.MsgOptionText(fmt.Sprintf("Alert: %s", e.LoggerName), true), slack.MsgOptionBlocks(
			blocks...,
		)); err != nil {
			return stackerr.Wrap(err)
		}

		return nil
	}
}
