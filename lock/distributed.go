package lock

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync/atomic"
	"time"

	"github.com/Invicton-Labs/go-common/aws/lambda"
	"github.com/Invicton-Labs/go-common/collections"
	"github.com/Invicton-Labs/go-common/conversions"
	"github.com/Invicton-Labs/go-common/dateutils"
	"github.com/Invicton-Labs/go-common/gensync"
	"github.com/Invicton-Labs/go-common/log"
	"github.com/Invicton-Labs/go-common/slack/links"
	"github.com/Invicton-Labs/go-stackerr"
	"github.com/aws/aws-lambda-go/lambdacontext"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/arn"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/google/uuid"
	"golang.org/x/sync/errgroup"
)

const (
	metaColumn     string = "Metadata"
	acquiredColumn string = "AcquiredUnixNano"
	logsUrlColumn  string = "LogsUrl"
	expiresColumn  string = "ExpiresUnixNano"
)

var (
	runId          string
	lockCounterMap gensync.Map[string, *atomic.Int32]
)

func init() {
	runId = strings.ToLower(uuid.NewString())
}

type DistributedLockerConfig struct {
	TableArn      string `json:"arn"`
	KeyColumn     string `json:"key_column"`
	VersionColumn string `json:"version_column"`
	// OPTIONAL. An AWS config to use. If not provided,
	// the default config will be used.
	AwsConfig *aws.Config
}

type LockData interface {
	// Gets the key for the lock
	Key() string
	// Gets the version for the lock
	Version() string
	// Gets the time the lock was acquired
	Acquired() time.Time
	// Gets the time the lock will expire. Only provided
	// for getting existing locks (will be zero-value when
	// used on a lock that is held by this process)
	Expires() time.Time
	// Gets the CloudWatch Logs URL for the entity that holds
	// the lock
	LogsUrl() string
	// Gets the metadata that was set for the lock
	Metadata() map[string]json.RawMessage
	// Active checks whether the lock is currently active (held by something).
	Active() bool
}

type lockData struct {
	key      string
	version  string
	acquired time.Time
	expires  time.Time
	logsUrl  string
	metadata map[string]json.RawMessage
	active   bool
}

func (ld lockData) Key() string {
	return ld.key
}

func (ld lockData) Version() string {
	return ld.version
}
func (ld lockData) Acquired() time.Time {
	return ld.acquired
}
func (ld lockData) Expires() time.Time {
	return ld.expires
}
func (ld lockData) LogsUrl() string {
	return ld.logsUrl
}
func (ld lockData) Metadata() map[string]json.RawMessage {
	return ld.metadata
}
func (ld lockData) Active() bool {
	return ld.active
}

// DistributedLock is a lock that can be used across multiple processes, computers, etc.
// It requires internet connectivity and an AWS DynamoDB table to use.
type DistributedLock interface {
	/*
		Unlock will release this lock.

		Arguments:

		ctx - the context to use for all operations in the unlock. If the context is cancelled,
		the unlock attempt will be cancelled and the expiry will eventually expire.

		Return Values:

		err - an error generated by this function
	*/
	Unlock(ctx context.Context) (err stackerr.Error)

	// Include all of the functions for lock data
	LockData
}

type distributedLock struct {
	distributedLocker *distributedLocker
	unlockCtxCancel   context.CancelFunc
	locked            atomic.Bool
	heartbeatErrGroup errgroup.Group

	// Include the lock data
	lockData
}

func (dl *distributedLock) Unlock(ctx context.Context) stackerr.Error {
	// Cancel the context for the heartbeat
	dl.unlockCtxCancel()

	// Wait for the heartbeat to finish (it should exit now that the
	// context has been cancelled).
	heartbeatErr := stackerr.Wrap(dl.heartbeatErrGroup.Wait())

	if _, err := dl.distributedLocker.client.UpdateItem(ctx, &dynamodb.UpdateItemInput{
		TableName: &dl.distributedLocker.tableName,
		Key: map[string]types.AttributeValue{
			dl.distributedLocker.config.KeyColumn: &types.AttributeValueMemberS{
				Value: dl.key,
			},
		},
		// Update the expires time
		UpdateExpression: conversions.GetPtr("SET #expires_column = :expires_unix_nano"),
		// Only update it if we still hold the lock
		ConditionExpression: conversions.GetPtr("#version_column = :version"),
		ExpressionAttributeNames: map[string]string{
			"#expires_column": expiresColumn,
			"#version_column": dl.distributedLocker.config.VersionColumn,
		},
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":expires_unix_nano": &types.AttributeValueMemberN{
				Value: fmt.Sprintf("%d", time.Now().UnixNano()),
			},
			":version": &types.AttributeValueMemberS{
				Value: dl.version,
			},
		},
		ReturnConsumedCapacity: types.ReturnConsumedCapacityNone,
		ReturnValues:           types.ReturnValueNone,
	}); err != nil {
		var ccfe *types.ConditionalCheckFailedException
		if errors.As(err, &ccfe) {
			return stackerr.Errorf("could not unlock distributed lock '%s', as it is not currently locked by this process", dl.key)
		}
		return stackerr.Wrap(err)
	}

	return heartbeatErr
}

type DistributedLocker interface {
	/*
		Lock will attempt to acquire a distributed lock for the given key.

		Arguments:

		ctx - the context to use for all operations in the lock. If the context is cancelled,
		the lock will not be updated and the lock will eventually expire.

		key - the key of the distributed lock to acquire.

		metadata - a map of metadata that should be stored with the lock. This can be useful
		for debugging or logging when a lock cannot be acquired because it's already held.

		Return Values:

		newCtx - a context that will be cancelled if the heartbeat fails (but will NOT be
		cancelled if the heartbeat exits because the lock was intentionally released).

		newLock - the new lock that was acquired, if it could be acquried. If not, this will be nil.

		existingLock - the existing lock, if there already was one. If a new lock was acquired,
		this will be nil.

		err - an error generated by this function
	*/
	Lock(ctx context.Context, key string, metadata map[string]any) (newCtx context.Context, newLock DistributedLock, existingLock LockData, err stackerr.Error)

	// GetAllLocks will get a map of all locks that are stored in the lock table, regardless of whether they're active
	GetAllLocks(ctx context.Context) (map[string]LockData, stackerr.Error)

	// GetActiveLocks will get a map of all active locks
	GetActiveLocks(ctx context.Context) (map[string]LockData, stackerr.Error)

	// GetExpiredLocks will get a map of all expired locks
	GetExpiredLocks(ctx context.Context) (map[string]LockData, stackerr.Error)
}

type distributedLocker struct {
	client    *dynamodb.Client
	config    DistributedLockerConfig
	tableName string
}

func (dl *distributedLocker) parseLockData(item map[string]types.AttributeValue) (LockData, stackerr.Error) {
	var key, version, logsUrl string
	var acquiredUnixNano, expiresUnixNano int64
	metadata := map[string]json.RawMessage{}

	// Extract the key column value
	keyValue, ok := item[dl.config.KeyColumn]
	if !ok {
		return nil, stackerr.Errorf("No key field in existing lock row")
	}
	// Try to unmarshal it
	if err := attributevalue.Unmarshal(keyValue, &key); err != nil {
		return nil, stackerr.Errorf("Version field in existing lock row is not of expected type")
	}

	// Extract the version column value
	versionValue, ok := item[dl.config.VersionColumn]
	if !ok {
		return nil, stackerr.Errorf("No version field in existing lock row")
	}
	// Try to unmarshal it
	if err := attributevalue.Unmarshal(versionValue, &version); err != nil {
		return nil, stackerr.Errorf("Version field in existing lock row is not of expected type")
	}

	// Extract the acquired column value
	acquiredValue, ok := item[acquiredColumn]
	if !ok {
		return nil, stackerr.Errorf("No acquired field in existing lock row")
	}
	// Try to unmarshal it
	if err := attributevalue.Unmarshal(acquiredValue, &acquiredUnixNano); err != nil {
		return nil, stackerr.Errorf("Acquired field in existing lock row is not of expected type")
	}

	// Extract the expires column value
	expiresValue, ok := item[expiresColumn]
	if !ok {
		return nil, stackerr.Errorf("No expires field in existing lock row")
	}
	// Try to unmarshal it
	if err := attributevalue.Unmarshal(expiresValue, &expiresUnixNano); err != nil {
		return nil, stackerr.Errorf("Expires field in existing lock row is not of expected type")
	}

	// Check if there's a logs URL
	if logsUrlValue, ok := item[logsUrlColumn]; ok {
		// If there is, try to unmarshal it
		if err := attributevalue.Unmarshal(logsUrlValue, &logsUrl); err != nil {
			// Log the error, but don't exit out since it doesn't prevent us from continuing
			// (it just makes the log alerts a bit less useful)
			log.Errorf("Logs URL field in existing lock row is not of expected type")
		}
	}

	// Check if there's metadata
	if metadataValue, ok := item[metaColumn]; ok {
		var rawMetadata string
		// If there is, try to unmarshal it
		if err := attributevalue.Unmarshal(metadataValue, &rawMetadata); err != nil {
			log.Errorf("Metadata field in existing lock row is not of expected type")
		} else {
			if err := json.Unmarshal([]byte(rawMetadata), &metadata); err != nil {
				log.With(
					"json", rawMetadata,
				).Errorf("Metadata field in existing lock row is not in valid JSON format")
			}
		}
	}

	acquired := dateutils.TimeFromUnix(acquiredUnixNano)
	expires := dateutils.TimeFromUnix(expiresUnixNano)

	return lockData{
		key:      key,
		version:  version,
		acquired: acquired,
		expires:  expires,
		logsUrl:  logsUrl,
		metadata: metadata,
		active:   expires.After(time.Now()),
	}, nil
}

func (dl *distributedLocker) getExistingLock(ctx context.Context, key string) (LockData, stackerr.Error) {
	keyAttribute, err := attributevalue.Marshal(key)
	if err != nil {
		return nil, stackerr.Wrap(err)
	}

	// Get the existing lock
	existing, err := dl.client.GetItem(ctx, &dynamodb.GetItemInput{
		TableName: &dl.tableName,
		Key: map[string]types.AttributeValue{
			dl.config.KeyColumn: keyAttribute,
		},
		// Ensure the lock read is consistent
		ConsistentRead: conversions.GetPtr(true),
	})
	if err != nil {
		return nil, stackerr.Wrap(err)
	}

	existingLockData, serr := dl.parseLockData(existing.Item)
	if serr != nil {
		return nil, serr.WithSingle("key", key)
	}

	return existingLockData, nil
}

func (dl *distributedLocker) Lock(ctx context.Context, key string, metadata map[string]any) (newCtx context.Context, newLock DistributedLock, existingLock LockData, err stackerr.Error) {

	// How long we hold the lock for on each heartbeat
	lockDuration := 20 * time.Second
	// Heartbeats occur at half of the lock duration. This ensures we always
	// keep it locked.
	heartbeatInterval := lockDuration / 2

	// The initial expiry time is now plus the lock duration
	initialExpiry := time.Now().Add(lockDuration)

	acquiredUnixNano := time.Now().UnixNano()

	// The lock row values to insert
	attributes := map[string]types.AttributeValue{
		// Put the key name in the key column
		dl.config.KeyColumn: &types.AttributeValueMemberS{
			Value: key,
		},
		// Set the timestamp for when the lock was most recently acquired
		acquiredColumn: &types.AttributeValueMemberN{
			Value: fmt.Sprintf("%d", acquiredUnixNano),
		},
		// Set the timestamp for when the lock should expire
		expiresColumn: &types.AttributeValueMemberN{
			Value: fmt.Sprintf("%d", initialExpiry.UnixNano()),
		},
	}

	var lockerId, logsUrl string
	// Use the AWS request ID if available
	if lc, ok := lambdacontext.FromContext(ctx); ok {
		lockerId = lc.AwsRequestID
		// Try to generate the URL for accessing the log stream. This
		// is useful for external alerts (e.g. Slack notifications of errors)
		// so the logs can be quickly and easily accessed.
		logsUrl, err = lambda.RequestIdLogStreamUrlFromContext(ctx)
		if err != nil {
			return nil, nil, nil, err
		}
		attributes[logsUrlColumn] = &types.AttributeValueMemberS{
			Value: logsUrl,
		}
	} else {
		lockerId = "local/" + runId
	}

	lockCounter, _ := lockCounterMap.LoadOrStore(key, &atomic.Int32{})
	lockCount := lockCounter.Add(1) - 1

	version := fmt.Sprintf("%s-%d", lockerId, lockCount)

	// Sweeten the logger with useful data
	log := log.With(
		"lock_key", key,
		"lock_version", version,
	)

	// Set the version in the version column
	attributes[dl.config.VersionColumn] = &types.AttributeValueMemberS{
		Value: version,
	}

	// If no metadata was provided, create an empty map for it, for consistency
	if metadata == nil {
		metadata = map[string]any{}
	}

	// Marshal the metadata
	meta, cerr := json.Marshal(metadata)
	if cerr != nil {
		return nil, nil, nil, stackerr.Wrap(cerr)
	}
	// Set the metadata JSON into the metadata column
	attributes[metaColumn] = &types.AttributeValueMemberS{
		Value: string(meta),
	}

	// Put an item for the lock, where either the row does not exist,
	// or the lock has expired.
	if _, err := dl.client.PutItem(ctx, &dynamodb.PutItemInput{
		TableName:           &dl.tableName,
		Item:                attributes,
		ConditionExpression: conversions.GetPtr("attribute_not_exists(#key_column) OR #expires_column <= :current_time_nano"),
		ExpressionAttributeNames: map[string]string{
			"#key_column":     dl.config.KeyColumn,
			"#expires_column": expiresColumn,
		},
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":current_time_nano": &types.AttributeValueMemberN{
				Value: fmt.Sprintf("%d", time.Now().UnixNano()),
			},
		},
		ReturnValues: types.ReturnValueNone,
	}); err != nil {
		// There was an error updating it. Check if it was a conditional check failure.
		// If it was, that means that there's already a lock that isn't expired.
		var ccfe *types.ConditionalCheckFailedException
		if errors.As(err, &ccfe) {

			// Try to get the existing lock
			existingLock, err := dl.getExistingLock(ctx, key)
			if err != nil {
				return ctx, nil, nil, err
			}

			// Log that we failed to acquire the lock
			log.Infow("Distributed lock acquisition failed, lock already held",
				"existing_lock_version", existingLock.Version(),
				"existing_lock_acquired", existingLock.Acquired(),
				"existing_lock_active", existingLock.Active(),
				"existing_lock_logs", existingLock.LogsUrl(),
			)
			return ctx, nil, existingLock, nil
		}

		// It was an error other than an existing lock, so return that error
		return ctx, nil, nil, stackerr.Wrap(err)
	}

	// Log that we succeeded in acquiring the lock
	log.Infow("Distributed lock acquired")

	// Convert the metadata input to match the output type when getting an existing lock
	metadataJson, err := collections.TransformMapWithErr(metadata, func(key string, value any) (transformedKey string, transformedValue json.RawMessage, err stackerr.Error) {
		j, cerr := json.Marshal(value)
		if cerr != nil {
			return "", nil, stackerr.Wrap(err)
		}
		return key, j, nil
	})
	if err != nil {
		return ctx, nil, nil, err
	}

	// This is a context that can be cancelled when the lock is released,
	// which will lead to a clean exit of the heartbeat.
	unlockCtx, unlockCtxCancel := context.WithCancel(ctx)

	// Create the passthrough context, which gets cancelled if the heartbeat
	// exits with an error.
	passthroughCtx, passthroughCtxCancel := context.WithCancel(ctx)

	lock := distributedLock{
		distributedLocker: dl,
		unlockCtxCancel:   unlockCtxCancel,
		locked:            atomic.Bool{},
		heartbeatErrGroup: errgroup.Group{},
		lockData: lockData{
			key:      key,
			version:  version,
			acquired: dateutils.TimeFromUnix(acquiredUnixNano),
			logsUrl:  logsUrl,
			metadata: metadataJson,
			active:   true,
		},
	}
	lock.locked.Store(true)

	// Start the heartbeat routine
	lock.heartbeatErrGroup.Go(func() (err error) {
		defer func() {
			// Recover any panics and convert them to stack errors
			if r := recover(); r != nil {
				err = stackerr.FromRecover(r)
			}

			// If the heartbeat exited with an error, cancel
			// the passthrough context so that downstream
			// processes know that we lost the lock.
			if err != nil {
				passthroughCtxCancel()
			}

			// Ensure the returned error is a stack error
			err = stackerr.WrapWithoutExtraStack(err)
		}()

		for {
			timer := time.NewTimer(heartbeatInterval)
			select {
			case <-unlockCtx.Done():

				// Stop and clear the heartbeat timer
				if !timer.Stop() {
					<-timer.C
				}

				if lock.locked.Load() {
					// The context was cancelled but the lock is still held,
					// so that's an error.
					return stackerr.Wrap(ctx.Err())
				}

				// The lock is no longer held, so the context was cancelled by (or after)
				// the lock being unlocked
				return nil

			// Wait for the heartbeat interval
			case <-timer.C:
				log.Debugw("Distributed lock heartbeat")

				// Renew the expiry on the lock we hold
				if _, err := dl.client.UpdateItem(ctx, &dynamodb.UpdateItemInput{
					TableName: &dl.tableName,
					Key: map[string]types.AttributeValue{
						dl.config.KeyColumn: &types.AttributeValueMemberS{
							Value: key,
						},
					},
					// Update the expiry time
					UpdateExpression: conversions.GetPtr("SET #expires_column = :expires_time_nano"),
					// Only update it if we still hold the lock
					ConditionExpression: conversions.GetPtr("#version_column = :version"),
					ExpressionAttributeNames: map[string]string{
						"#expires_column": expiresColumn,
						"#version_column": dl.config.VersionColumn,
					},
					ExpressionAttributeValues: map[string]types.AttributeValue{
						":expires_time_nano": &types.AttributeValueMemberN{
							Value: fmt.Sprintf("%d", time.Now().Add(heartbeatInterval).UnixNano()),
						},
						":version": &types.AttributeValueMemberS{
							Value: version,
						},
					},
					ReturnConsumedCapacity: types.ReturnConsumedCapacityNone,
					ReturnValues:           types.ReturnValueNone,
				}); err != nil {
					// The update failed. Check if it was a conditional check failure.
					var ccfe *types.ConditionalCheckFailedException
					if errors.As(err, &ccfe) {
						// It was a conditional check failure, so get the existing lock that
						// caused it to fail. Store it in a variable that is accessible
						// to the deferre
						existingLock, err := dl.getExistingLock(ctx, key)
						if err != nil {
							return err
						}
						err = stackerr.Errorf("Distributed lock has been lost").With(map[string]any{
							"existing_lock_version":  existingLock.Version(),
							"existing_lock_acquired": existingLock.Acquired(),
							"existing_lock_active":   existingLock.Active(),
							"existing_lock_logs":     links.NewSlackLink(existingLock.LogsUrl(), "Log Stream"),
						})
						log.Error(err)
						return err
					}
					return stackerr.Wrap(err)
				}
			}
		}
	})

	return passthroughCtx, &lock, nil, nil
}

func (dl *distributedLocker) GetAllLocks(ctx context.Context) (map[string]LockData, stackerr.Error) {
	return dl.getLocks(ctx, all)
}

func (dl *distributedLocker) GetActiveLocks(ctx context.Context) (map[string]LockData, stackerr.Error) {
	return dl.getLocks(ctx, active)
}

func (dl *distributedLocker) GetExpiredLocks(ctx context.Context) (map[string]LockData, stackerr.Error) {
	return dl.getLocks(ctx, expired)
}

type lockType int

const (
	all lockType = iota
	active
	expired
)

func (dl *distributedLocker) getLocks(ctx context.Context, typ lockType) (map[string]LockData, stackerr.Error) {

	input := &dynamodb.ScanInput{
		TableName:      &dl.tableName,
		Select:         types.SelectAllAttributes,
		ConsistentRead: conversions.GetPtr(true),
	}

	if typ != all {
		input.ExpressionAttributeNames = map[string]string{
			"#expires_column": expiresColumn,
		}
		input.ExpressionAttributeValues = map[string]types.AttributeValue{
			":current_time_unix_nano": &types.AttributeValueMemberN{
				Value: fmt.Sprintf("%d", time.Now().UnixNano()),
			},
		}

		switch typ {
		case active:
			// Only get items where the expires value is in the future
			input.FilterExpression = conversions.GetPtr("#expires_column > :current_time_unix_nano")
		case expired:
			// Only get items where the expires value is now or in the past
			input.FilterExpression = conversions.GetPtr("#expires_column <= :current_time_unix_nano")
		}
	}

	paginator := dynamodb.NewScanPaginator(dl.client, input)

	items := []map[string]types.AttributeValue{}

	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, stackerr.Wrap(err)
		}
		items = append(items, page.Items...)
	}

	locks, err := collections.TransformSliceToMapWithErr(items, func(_ int, sliceValue map[string]types.AttributeValue) (mapKey string, mapValue LockData, err stackerr.Error) {
		lock, err := dl.parseLockData(sliceValue)
		if err != nil {
			return "", nil, err
		}
		return lock.Key(), lock, nil
	})
	if err != nil {
		return nil, err
	}

	return locks, nil
}

// NewDistributedLocker creates a new DynamoDB-based distributed locker that allows holding global locks.
func NewDistributedLocker(ctx context.Context, dlConfig DistributedLockerConfig) (DistributedLocker, stackerr.Error) {
	if dlConfig.TableArn == "" {
		return nil, stackerr.Errorf("the `config.TableArn` field must not be empty")
	}
	if dlConfig.KeyColumn == "" {
		return nil, stackerr.Errorf("the `config.KeyColumn` field must not be empty")
	}

	a, cerr := arn.Parse(dlConfig.TableArn)
	if cerr != nil {
		return nil, stackerr.Wrap(cerr)
	}

	var cfg aws.Config
	if dlConfig.AwsConfig != nil {
		cfg = *dlConfig.AwsConfig
	} else {
		var err error
		// Get the config for our AWS credentials
		cfg, err = config.LoadDefaultConfig(ctx)
		if err != nil {
			return nil, stackerr.Wrap(err)
		}
		// Set the region to match the DynamoDB table's region
		cfg.Region = a.Region
	}

	// Create an SQS client
	client := dynamodb.NewFromConfig(cfg)

	return &distributedLocker{
		client:    client,
		config:    dlConfig,
		tableName: strings.TrimPrefix(a.Resource, "table/"),
	}, nil
}
