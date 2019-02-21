package main

type Config struct {
    Http HttpConfig
    Https HttpsConfig
    Cognito Cognito
}

type HttpConfig struct {
    Port string
}

type HttpsConfig struct {
    Port string
    CertificateFile string
    PrivateKeyFile string
}

type Cognito struct {
    UserPoolId string
    ClientId string
}
