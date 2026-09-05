package main

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/umputun/go-flags"
)

func TestProxyTrustHeadersOption(t *testing.T) {
	for _, tt := range []struct {
		name string
		args []string
		env  string
		want bool
	}{
		{"default", nil, "", false},
		{"flag", []string{"--proxy-trust-headers"}, "", true},
		{"environment", nil, "true", true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("PROXY_TRUST_HEADERS", tt.env)
			if tt.env == "" {
				require.NoError(t, os.Unsetenv("PROXY_TRUST_HEADERS"))
			}
			cfg := opts
			args := append([]string{"--key=test-signing-key", "--domain=localhost"}, tt.args...)
			_, err := flags.NewParser(&cfg, flags.None).ParseArgs(args)
			require.NoError(t, err)
			assert.Equal(t, tt.want, cfg.ProxyTrustHeaders)
		})
	}
}
