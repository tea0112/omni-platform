package main

import (
	_ "github.com/go-chi/chi/v5"
	_ "connectrpc.com/connect"
	_ "go.uber.org/fx"
	_ "github.com/jackc/pgx/v5/stdlib"
	_ "github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-jwt/jwt/v5"
	_ "github.com/spf13/viper"
	_ "go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	_ "github.com/stretchr/testify/assert"
	_ "github.com/google/uuid"
	_ "golang.org/x/crypto/bcrypt"
	_ "google.golang.org/protobuf/proto"
	_ "google.golang.org/grpc"
)

func main() {}
