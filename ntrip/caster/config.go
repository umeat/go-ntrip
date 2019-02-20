package caster

type Config struct {
    Http HttpConfig
    Https HttpsConfig
}

type HttpConfig struct {
    Port string
}

type HttpsConfig struct {
    Port string
    CertificateFile string
    PrivateKeyFile string
}
