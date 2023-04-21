package cognito

import (
	"context"
	"errors"

	"github.com/Invicton-Labs/go-stackerr"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/cognitoidentityprovider"
	"github.com/aws/aws-sdk-go-v2/service/cognitoidentityprovider/types"
)

var cognitoClient *cognitoidentityprovider.Client

func getCognitoClient(ctx context.Context) *cognitoidentityprovider.Client {
	if cognitoClient != nil {
		return cognitoClient
	}
	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		panic(err)
	}
	cognitoClient = cognitoidentityprovider.NewFromConfig(cfg)
	return cognitoClient
}

type CognitoUser struct {
	Username   string
	Attributes map[string]string
}

func GetCognitoUserByUsername(ctx context.Context, userPoolId string, username string) (*CognitoUser, stackerr.Error) {
	client := getCognitoClient(ctx)
	cUser, err := client.AdminGetUser(context.TODO(), &cognitoidentityprovider.AdminGetUserInput{UserPoolId: &userPoolId, Username: &username})
	if err != nil {
		var oe *types.UserNotFoundException
		if errors.As(err, &oe) {
			return nil, nil
		}
		return nil, stackerr.Wrap(err)
	}
	user := &CognitoUser{
		Username:   *cUser.Username,
		Attributes: make(map[string]string, len(cUser.UserAttributes)),
	}
	for _, att := range cUser.UserAttributes {
		user.Attributes[*att.Name] = *att.Value
	}
	return user, nil
}
