package authorizers

import (
    "github.com/aws/aws-sdk-go/aws"
    "github.com/aws/aws-sdk-go/aws/session"
    "github.com/aws/aws-sdk-go/service/cognitoidentityprovider"
    "github.com/umeat/go-ntrip/ntrip/caster"
    "crypto/hmac"
    "crypto/sha256"
    "encoding/base64"
    "errors"

    "github.com/dgrijalva/jwt-go"
    "fmt"
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

func NewCognitoAuthorizer(userPoolId, clientId string) (auth Cognito, err error) {
    // TODO: Load from config - not secret, using AWS credentials for secret
    auth.UserPoolId = userPoolId
    auth.ClientId = clientId

    auth.Cip = cognitoidentityprovider.New(session.Must(session.NewSession()))
    return auth, err
}

func (auth Cognito) Authenticate(conn *caster.Connection) (err error) {
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

    resp, err := auth.Cip.AdminInitiateAuth(params) // TODO: Inspect response for claims and implement path based auth
    if err != nil {
        return err
    }

    token, _ := jwt.Parse(*resp.AuthenticationResult.IdToken, nil)
    fmt.Println(token.Claims)

    // Not sure if it makes sense to return the ID token in a header
    // Usually you would have the auth endpoint be elsewhere and return the token in the body of the response, but we don't really have the luxury of palming it off
    //conn.Writer.Header().Set("Authorization", "Bearer " + *resp.AuthenticationResult.IdToken) 
    return nil
}
