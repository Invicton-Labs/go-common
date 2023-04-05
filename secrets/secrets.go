package secrets

import (
	"context"
	"encoding/base64"

	"github.com/Invicton-Labs/go-stackerr"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
)

var secretsClient *secretsmanager.Client

func getSecretsManagerClient(ctx context.Context) *secretsmanager.Client {
	if secretsClient != nil {
		return secretsClient
	}
	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		panic(err)
	}
	secretsClient = secretsmanager.NewFromConfig(cfg)
	return secretsClient
}

// GetSecret gets a general secret string from AWS SecretsManager
func GetSecret(ctx context.Context, arnString string) (*string, stackerr.Error) {
	client := getSecretsManagerClient(ctx)
	result, err := client.GetSecretValue(context.TODO(), &secretsmanager.GetSecretValueInput{
		SecretId:     aws.String(arnString),
		VersionStage: aws.String("AWSCURRENT"),
	})
	if err != nil {
		return nil, stackerr.Wrap(err)
	}

	// Decrypts secret using the associated KMS CMK.
	// Depending on whether the secret is a string or binary, one of these fields will be populated.
	var secretString *string
	if result.SecretString != nil {
		secretString = result.SecretString
	} else {
		decodedBinarySecretBytes := make([]byte, base64.StdEncoding.DecodedLen(len(result.SecretBinary)))
		len, err := base64.StdEncoding.Decode(decodedBinarySecretBytes, result.SecretBinary)
		if err != nil {
			return nil, stackerr.Wrap(err)
		}
		secretString = aws.String(string(decodedBinarySecretBytes[:len]))
	}
	return secretString, nil
}
