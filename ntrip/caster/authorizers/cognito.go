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

func (auth Cognito) Authorize(conn *caster.Connection) (err error) {
    switch conn.Request.Method {
    case "GET":
        return nil // TODO: Implement list of Closed mountpoints for which a client needs authorized access

    case "POST":
        username, password, auth_provided := conn.Request.BasicAuth()
        if !auth_provided {
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

        resp, err := auth.Cip.AdminInitiateAuth(params)
        if err != nil {
            return err
        }

        token, _ := jwt.Parse(*resp.AuthenticationResult.IdToken, nil)
        if groups, exists := token.Claims.(jwt.MapClaims)["cognito:groups"]; exists {
            for _, group := range groups.([]interface{}) {
                if group == "mount:" + conn.Request.URL.Path[1:] {
                    return nil
                }
            }
        }

        return errors.New("Not authorized for Mountpoint")
    }

    // Not sure if it makes sense to return the ID token in a header
    // Usually you would have the auth endpoint be elsewhere and return the token in the body of the response, but we don't really have the luxury of palming it off
    //conn.Writer.Header().Set("Authorization", "Bearer " + *resp.AuthenticationResult.IdToken) 
    return errors.New("Method not implemented")
}
