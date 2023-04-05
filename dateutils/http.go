package dateutils

import (
	"net/http"
	"time"

	"github.com/Invicton-Labs/go-stackerr"
	awstime "github.com/aws/smithy-go/time"
)

// HttpHeaderDate gets a time.Time from a given header key in the given header object.
func HttpHeaderDate(headers http.Header, key string) (time.Time, stackerr.Error) {
	str := headers.Get(key)
	if str == "" {
		return time.Time{}, stackerr.Errorf("Failed to find header with name '%s'", key)
	}
	dt, cerr := awstime.ParseHTTPDate(str)
	if cerr != nil {
		return time.Time{}, stackerr.Wrap(cerr)
	}
	return dt, nil
}
