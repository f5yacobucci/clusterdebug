package config

import (
	"fmt"
	"math/rand"
	"os"
	"strconv"
	"time"

	"github.com/labstack/echo/v4"
)

type (
	Config struct {
		Name      string
		IP        string
		Namespace string
		Service   string
		Domain    string
		Port      uint16
	}

	ClusterContext struct {
		echo.Context
		conf Config
	}
)

const (
	unknown = "debug-unknown"
)

func NewClusterContext(ctx echo.Context, c Config) *ClusterContext {
	return &ClusterContext{
		ctx,
		c,
	}
}

func (c *ClusterContext) Config() Config {
	return c.conf
}

func NewConfig() Config {
	config := Config{}

	c, ok := os.LookupEnv("STATEFUL_PORT")
	if ok && len(c) > 0 {
		value, err := strconv.ParseUint(c, 10, 16)
		if err == nil {
			config.Port = uint16(value)
		}
	} else {
		config.Port = 8080
	}

	c, ok = os.LookupEnv("SELF_NAME")
	if ok && len(c) > 0 {
		config.Name = c
	} else {
		rand.Seed(time.Now().UnixNano())
		config.Name = fmt.Sprintf("debug-%d", rand.Intn(4294967296))
	}

	c, ok = os.LookupEnv("SELF_IP")
	if ok && len(c) > 0 {
		config.IP = c
	} else {
		config.IP = unknown
	}

	c, ok = os.LookupEnv("SELF_NAMESPACE")
	if ok && len(c) > 0 {
		config.Namespace = c
	} else {
		config.Namespace = unknown
	}

	c, ok = os.LookupEnv("SELF_SERVICE")
	if ok && len(c) > 0 {
		config.Service = c
	} else {
		config.Service = unknown
	}

	config.Domain = fmt.Sprintf("%s.%s.svc.cluster.local", config.Service, config.Namespace)

	return config
}
