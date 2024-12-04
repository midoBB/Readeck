// SPDX-FileCopyrightText: © 2024 Olivier Meunier <olivier@neokraft.net>
//
// SPDX-License-Identifier: AGPL-3.0-only

package configs

import (
	"fmt"
	"log/slog"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGenerateKey(t *testing.T) {
	lengths := [][2]int{
		{8, 10},
		{10, 20},
	}

	for _, test := range lengths {
		t.Run(fmt.Sprintf("%d - %d", test[0], test[1]), func(t *testing.T) {
			k := GenerateKey(test[0], test[1])
			require.GreaterOrEqual(t, len(k), test[0])
			require.LessOrEqual(t, len(k), test[1])
		})
	}

	t.Run("error", func(t *testing.T) {
		require.Panics(t, func() {
			_ = GenerateKey(5, 5)
		})
	})
}

func TestEnvVars(t *testing.T) {
	cf := config{}

	tests := []struct {
		name   string
		value  string
		expect func(*require.Assertions, error)
	}{
		{"READECK_LOG_LEVEL", "warn", func(assert *require.Assertions, err error) {
			assert.NoError(err)
			assert.Equal(slog.LevelWarn, cf.Main.LogLevel)
		}},
		{"READECK_DEV_MODE", "1", func(assert *require.Assertions, err error) {
			assert.NoError(err)
			assert.True(cf.Main.DevMode)
		}},
		{"READECK_DEV_MODE", "0", func(assert *require.Assertions, err error) {
			assert.NoError(err)
			assert.False(cf.Main.DevMode)
		}},
		{"READECK_DEV_MODE", "abc", func(assert *require.Assertions, err error) {
			assert.ErrorContains(err, "invalid syntax")
		}},
		{"READECK_SERVER_HOST", "localhost", func(assert *require.Assertions, err error) {
			assert.NoError(err)
			assert.Equal("localhost", cf.Server.Host)
		}},
		{"READECK_SERVER_PORT", "8000", func(assert *require.Assertions, err error) {
			assert.NoError(err)
			assert.Equal(8000, cf.Server.Port)
		}},
		{"READECK_SERVER_PREFIX", "/app", func(assert *require.Assertions, err error) {
			assert.NoError(err)
			assert.Equal("/app", cf.Server.Prefix)
		}},
		{"READECK_TRUSTED_PROXIES", "127.0.0.2,192.168.0.1/26,fd00:abcd::/64", func(assert *require.Assertions, err error) {
			assert.NoError(err)
			r := []string{}
			for _, x := range cf.Server.TrustedProxies {
				r = append(r, x.String())
			}
			assert.Equal([]string{"127.0.0.2/32", "192.168.0.0/26", "fd00:abcd::/64"}, r)

			v, exists := os.LookupEnv("READECK_TRUSTED_PROXIES")
			assert.Equal("", v)
			assert.False(exists)
		}},
		{"READECK_ALLOWED_HOSTS", "example.net,example.com", func(assert *require.Assertions, err error) {
			assert.NoError(err)
			assert.Equal([]string{"example.net", "example.com"}, cf.Server.AllowedHosts)
		}},
		{"READECK_DATABASE_SOURCE", "sqlite3::memory", func(assert *require.Assertions, err error) {
			assert.NoError(err)
			assert.Equal("sqlite3::memory", cf.Database.Source)

			v, exists := os.LookupEnv("READECK_DATABASE_SOURCE")
			assert.Equal("", v)
			assert.False(exists)
		}},
		{"READECK_PUBLIC_SHARE_TTL", "48", func(assert *require.Assertions, err error) {
			assert.NoError(err)
			assert.Equal(48, cf.Bookmarks.PublicShareTTL)
		}},
		{"READECK_WORKER_DSN", "memory://", func(assert *require.Assertions, err error) {
			assert.NoError(err)
			assert.Equal("memory://", cf.Worker.DSN)

			v, exists := os.LookupEnv("READECK_WORKER_DSN")
			assert.Equal("", v)
			assert.False(exists)
		}},
		{"READECK_WORKER_NUMBER", "10", func(assert *require.Assertions, err error) {
			assert.NoError(err)
			assert.Equal(10, cf.Worker.NumWorkers)
		}},
		{"READECK_WORKER_START", "true", func(assert *require.Assertions, err error) {
			assert.NoError(err)
			assert.True(cf.Worker.StartWorker)
		}},
		{"READECK_METRICS_HOST", "::1", func(assert *require.Assertions, err error) {
			assert.NoError(err)
			assert.Equal("::1", cf.Metrics.Host)
		}},
		{"READECK_METRICS_PORT", "8002", func(assert *require.Assertions, err error) {
			assert.NoError(err)
			assert.Equal(8002, cf.Metrics.Port)
		}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Setenv(test.name, test.value)
			err := cf.LoadEnv()
			test.expect(require.New(t), err)
		})
	}
}
