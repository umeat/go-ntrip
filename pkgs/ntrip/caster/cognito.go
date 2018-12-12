package caster

// TODO: This module could be in a different package - maybe in with cmd/ntripcaster

import (
    "github.com/aws/aws-sdk-go/aws"
    "github.com/aws/aws-sdk-go/aws/session"
    "github.com/aws/aws-sdk-go/service/cognitoidentityprovider"
    "crypto/hmac"
    "crypto/sha256"
    "encoding/base64"
    "errors"
    "os"
)

func SecretHash(username, clientID, clientSecret string) string {
    mac := hmac.New(sha256.New, []byte(clientSecret))
    mac.Write([]byte(username + clientID))
    return base64.StdEncoding.EncodeToString(mac.Sum(nil))
}

type Cognito struct {
    UserPoolId string
    ClientId string
    Cip *cognitoidentityprovider.CognitoIdentityProvider
}

func NewCognitoAuthorizer() (auth Cognito, err error) {
    // TODO: Load from config - not secret, using AWS credentials for secret
    auth.UserPoolId = os.Getenv("COGNITO_USER_POOL_ID")
    auth.ClientId = os.Getenv("COGNITO_CLIENT_ID")

    auth.Cip = cognitoidentityprovider.New(session.Must(session.NewSession()))
    return auth, err
}

func (auth Cognito) Authenticate(conn *Connection) (err error) {
    username, password, ok := conn.Request.BasicAuth() // TODO: Implement Bearer auth
    if !ok {
        return errors.New("Basic auth not provided")
    }

    params := &cognitoidentityprovider.AdminInitiateAuthInput{
        AuthFlow: aws.String("ADMIN_NO_SRP_AUTH"),
        AuthParameters: map[string]*string{
            "USERNAME": aws.String(username),
            "PASSWORD": aws.String(password),
        },
        ClientId:   aws.String(auth.ClientId),
        UserPoolId: aws.String(auth.UserPoolId),
    }

    _, err = auth.Cip.AdminInitiateAuth(params) // TODO: Inspect response for claims and implement path based auth
    if err != nil {
        return err
    }

    // Not sure if it makes sense to return the ID token in a header
    // Usually you would have the auth endpoint be elsewhere and return the token in the body of the response, but we don't really have the luxury of palming it off
    //conn.Writer.Header().Set("Authorization", "Bearer " + *resp.AuthenticationResult.IdToken) 
    return nil
}
