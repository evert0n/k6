/*
 *
 * k6 - a next-generation load testing tool
 * Copyright (C) 2016 Load Impact
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU Affero General Public License as
 * published by the Free Software Foundation, either version 3 of the
 * License, or (at your option) any later version.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU Affero General Public License for more details.
 *
 * You should have received a copy of the GNU Affero General Public License
 * along with this program.  If not, see <http://www.gnu.org/licenses/>.
 *
 */

package lib

import (
	"crypto/tls"
	"encoding/json"
	"net"
	"os"
	"reflect"
	"testing"
	"time"

	"github.com/kelseyhightower/envconfig"
	"github.com/loadimpact/k6/stats"
	"github.com/stretchr/testify/assert"
	"gopkg.in/guregu/null.v3"
)

func TestOptions(t *testing.T) {
	t.Run("Paused", func(t *testing.T) {
		opts := Options{}.Apply(Options{Paused: null.BoolFrom(true)})
		assert.True(t, opts.Paused.Valid)
		assert.True(t, opts.Paused.Bool)
	})
	t.Run("VUs", func(t *testing.T) {
		opts := Options{}.Apply(Options{VUs: null.IntFrom(12345)})
		assert.True(t, opts.VUs.Valid)
		assert.Equal(t, int64(12345), opts.VUs.Int64)
	})
	t.Run("VUsMax", func(t *testing.T) {
		opts := Options{}.Apply(Options{VUsMax: null.IntFrom(12345)})
		assert.True(t, opts.VUsMax.Valid)
		assert.Equal(t, int64(12345), opts.VUsMax.Int64)
	})
	t.Run("Duration", func(t *testing.T) {
		opts := Options{}.Apply(Options{Duration: NullDurationFrom(2 * time.Minute)})
		assert.True(t, opts.Duration.Valid)
		assert.Equal(t, "2m0s", opts.Duration.String())
	})
	t.Run("Iterations", func(t *testing.T) {
		opts := Options{}.Apply(Options{Iterations: null.IntFrom(1234)})
		assert.True(t, opts.Iterations.Valid)
		assert.Equal(t, int64(1234), opts.Iterations.Int64)
	})
	t.Run("Stages", func(t *testing.T) {
		opts := Options{}.Apply(Options{Stages: []Stage{{Duration: NullDurationFrom(1 * time.Second)}}})
		assert.NotNil(t, opts.Stages)
		assert.Len(t, opts.Stages, 1)
		assert.Equal(t, 1*time.Second, time.Duration(opts.Stages[0].Duration.Duration))
	})
	t.Run("MaxRedirects", func(t *testing.T) {
		opts := Options{}.Apply(Options{MaxRedirects: null.IntFrom(12345)})
		assert.True(t, opts.MaxRedirects.Valid)
		assert.Equal(t, int64(12345), opts.MaxRedirects.Int64)
	})
	t.Run("InsecureSkipTLSVerify", func(t *testing.T) {
		opts := Options{}.Apply(Options{InsecureSkipTLSVerify: null.BoolFrom(true)})
		assert.True(t, opts.InsecureSkipTLSVerify.Valid)
		assert.True(t, opts.InsecureSkipTLSVerify.Bool)
	})
	t.Run("TLSCipherSuites", func(t *testing.T) {
		for suiteName, suiteID := range SupportedTLSCipherSuites {
			t.Run(suiteName, func(t *testing.T) {
				opts := Options{}.Apply(Options{TLSCipherSuites: &TLSCipherSuites{suiteID}})

				assert.NotNil(t, opts.TLSCipherSuites)
				assert.Len(t, *(opts.TLSCipherSuites), 1)
				assert.Equal(t, suiteID, (*opts.TLSCipherSuites)[0])
			})
		}
	})
	t.Run("TLSVersion", func(t *testing.T) {
		versions := TLSVersions{Min: tls.VersionSSL30, Max: tls.VersionTLS12}
		opts := Options{}.Apply(Options{TLSVersion: &versions})

		assert.NotNil(t, opts.TLSVersion)
		assert.Equal(t, opts.TLSVersion.Min, TLSVersion(tls.VersionSSL30))
		assert.Equal(t, opts.TLSVersion.Max, TLSVersion(tls.VersionTLS12))

		t.Run("JSON", func(t *testing.T) {
			t.Run("Object", func(t *testing.T) {
				var opts Options
				jsonStr := `{"tlsVersion":{"min":"ssl3.0","max":"tls1.2"}}`
				assert.NoError(t, json.Unmarshal([]byte(jsonStr), &opts))
				assert.Equal(t, &TLSVersions{
					Min: TLSVersion(tls.VersionSSL30),
					Max: TLSVersion(tls.VersionTLS12),
				}, opts.TLSVersion)

				t.Run("Roundtrip", func(t *testing.T) {
					data, err := json.Marshal(opts.TLSVersion)
					assert.NoError(t, err)
					assert.Equal(t, `{"min":"ssl3.0","max":"tls1.2"}`, string(data))
					var vers2 TLSVersions
					assert.NoError(t, json.Unmarshal(data, &vers2))
					assert.Equal(t, &vers2, opts.TLSVersion)
				})
			})
			t.Run("String", func(t *testing.T) {
				var opts Options
				jsonStr := `{"tlsVersion":"tls1.2"}`
				assert.NoError(t, json.Unmarshal([]byte(jsonStr), &opts))
				assert.Equal(t, &TLSVersions{
					Min: TLSVersion(tls.VersionTLS12),
					Max: TLSVersion(tls.VersionTLS12),
				}, opts.TLSVersion)
			})
			t.Run("Blank", func(t *testing.T) {
				var opts Options
				jsonStr := `{"tlsVersion":""}`
				assert.NoError(t, json.Unmarshal([]byte(jsonStr), &opts))
				assert.Equal(t, &TLSVersions{}, opts.TLSVersion)
			})
		})
	})
	t.Run("TLSAuth", func(t *testing.T) {
		tlsAuth := []*TLSAuth{
			{TLSAuthFields{
				Domains: []string{"example.com", "*.example.com"},
				Cert: "-----BEGIN CERTIFICATE-----\n" +
					"MIIBoTCCAUegAwIBAgIUQl0J1Gkd6U2NIMwMDnpfH8c1myEwCgYIKoZIzj0EAwIw\n" +
					"EDEOMAwGA1UEAxMFTXkgQ0EwHhcNMTcwODE1MTYxODAwWhcNMTgwODE1MTYxODAw\n" +
					"WjAQMQ4wDAYDVQQDEwV1c2VyMTBZMBMGByqGSM49AgEGCCqGSM49AwEHA0IABLaf\n" +
					"xEOmBHkzbqd9/0VZX/39qO2yQq2Gz5faRdvy38kuLMCV+9HYrfMx6GYCZzTUIq6h\n" +
					"8QXOrlgYTixuUVfhJNWjfzB9MA4GA1UdDwEB/wQEAwIFoDAdBgNVHSUEFjAUBggr\n" +
					"BgEFBQcDAQYIKwYBBQUHAwIwDAYDVR0TAQH/BAIwADAdBgNVHQ4EFgQUxmQiq5K3\n" +
					"KUnVME945Byt3Ysvkh8wHwYDVR0jBBgwFoAU3qEhcpRgpsqo9V+LFns9a+oZIYww\n" +
					"CgYIKoZIzj0EAwIDSAAwRQIgSGxnJ+/cLUNTzt7fhr/mjJn7ShsTW33dAdfLM7H2\n" +
					"z/gCIQDyVf8DePtxlkMBScTxZmIlMQdNc6+6VGZQ4QscruVLmg==\n" +
					"-----END CERTIFICATE-----",
				Key: "-----BEGIN EC PRIVATE KEY-----\n" +
					"MHcCAQEEIAfJeoc+XgcqmYV0b4owmofx0LXwPRqOPXMO+PUKxZSgoAoGCCqGSM49\n" +
					"AwEHoUQDQgAEtp/EQ6YEeTNup33/RVlf/f2o7bJCrYbPl9pF2/LfyS4swJX70dit\n" +
					"8zHoZgJnNNQirqHxBc6uWBhOLG5RV+Ek1Q==\n" +
					"-----END EC PRIVATE KEY-----",
			}, nil},
			{TLSAuthFields{
				Domains: []string{"sub.example.com"},
				Cert: "-----BEGIN CERTIFICATE-----\n" +
					"MIIBojCCAUegAwIBAgIUWMpVQhmGoLUDd2x6XQYoOOV6C9AwCgYIKoZIzj0EAwIw\n" +
					"EDEOMAwGA1UEAxMFTXkgQ0EwHhcNMTcwODE1MTYxODAwWhcNMTgwODE1MTYxODAw\n" +
					"WjAQMQ4wDAYDVQQDEwV1c2VyMTBZMBMGByqGSM49AgEGCCqGSM49AwEHA0IABBfF\n" +
					"85gu8fDbNGNlsrtnO+4HvuiP4IXA041jjGczD5kUQ8aihS7hg81tSrLNd1jgxkkv\n" +
					"Po+3TQjzniysiunG3iKjfzB9MA4GA1UdDwEB/wQEAwIFoDAdBgNVHSUEFjAUBggr\n" +
					"BgEFBQcDAQYIKwYBBQUHAwIwDAYDVR0TAQH/BAIwADAdBgNVHQ4EFgQUU0JfPCQb\n" +
					"2YpQZV4j1yiRXBa7J64wHwYDVR0jBBgwFoAU3qEhcpRgpsqo9V+LFns9a+oZIYww\n" +
					"CgYIKoZIzj0EAwIDSQAwRgIhANYDaM18sXAdkjybHccH8xTbBWUNpOYvoHhrGW32\n" +
					"Ov9JAiEA7QKGpm07tQl8p+t7UsOgZu132dHNZUtfgp1bjWfcapU=\n" +
					"-----END CERTIFICATE-----",
				Key: "-----BEGIN EC PRIVATE KEY-----\n" +
					"MHcCAQEEINVilD5qOBkSy+AYfd41X0QPB5N3Z6OzgoBj8FZmSJOFoAoGCCqGSM49\n" +
					"AwEHoUQDQgAEF8XzmC7x8Ns0Y2Wyu2c77ge+6I/ghcDTjWOMZzMPmRRDxqKFLuGD\n" +
					"zW1Kss13WODGSS8+j7dNCPOeLKyK6cbeIg==\n" +
					"-----END EC PRIVATE KEY-----",
			}, nil},
		}
		opts := Options{}.Apply(Options{TLSAuth: tlsAuth})
		assert.Equal(t, tlsAuth, opts.TLSAuth)

		t.Run("Roundtrip", func(t *testing.T) {
			optsData, err := json.Marshal(opts)
			assert.NoError(t, err)

			var opts2 Options
			assert.NoError(t, json.Unmarshal(optsData, &opts2))
			if assert.Len(t, opts2.TLSAuth, len(opts.TLSAuth)) {
				for i := 0; i < len(opts2.TLSAuth); i++ {
					assert.Equal(t, opts.TLSAuth[i].TLSAuthFields, opts2.TLSAuth[i].TLSAuthFields)
					cert, err := opts2.TLSAuth[i].Certificate()
					assert.NoError(t, err)
					assert.NotNil(t, cert)
				}
			}
		})
	})
	t.Run("NoConnectionReuse", func(t *testing.T) {
		opts := Options{}.Apply(Options{NoConnectionReuse: null.BoolFrom(true)})
		assert.True(t, opts.NoConnectionReuse.Valid)
		assert.True(t, opts.NoConnectionReuse.Bool)
	})

	t.Run("Hosts", func(t *testing.T) {
		opts := Options{}.Apply(Options{Hosts: map[string]net.IP{
			"test.loadimpact.com": net.ParseIP("192.0.2.1"),
		}})
		assert.NotNil(t, opts.Hosts)
		assert.NotEmpty(t, opts.Hosts)
		assert.Equal(t, "192.0.2.1", opts.Hosts["test.loadimpact.com"].String())
	})

	t.Run("Thresholds", func(t *testing.T) {
		opts := Options{}.Apply(Options{Thresholds: map[string]stats.Thresholds{
			"metric": {
				Thresholds: []*stats.Threshold{{}},
			},
		}})
		assert.NotNil(t, opts.Thresholds)
		assert.NotEmpty(t, opts.Thresholds)
	})
	t.Run("External", func(t *testing.T) {
		opts := Options{}.Apply(Options{External: map[string]interface{}{"a": 1}})
		assert.Equal(t, map[string]interface{}{"a": 1}, opts.External)
	})

	t.Run("JSON", func(t *testing.T) {
		data, err := json.Marshal(Options{})
		assert.NoError(t, err)
		var opts Options
		assert.NoError(t, json.Unmarshal(data, &opts))
		assert.Equal(t, Options{}, opts)
	})
}

func TestOptionsEnv(t *testing.T) {
	testdata := map[struct{ Name, Key string }]map[string]interface{}{
		{"Paused", "K6_PAUSED"}: {
			"":      null.Bool{},
			"true":  null.BoolFrom(true),
			"false": null.BoolFrom(false),
		},
		{"VUs", "K6_VUS"}: {
			"":    null.Int{},
			"123": null.IntFrom(123),
		},
		{"VUsMax", "K6_VUS_MAX"}: {
			"":    null.Int{},
			"123": null.IntFrom(123),
		},
		{"Duration", "K6_DURATION"}: {
			"":    NullDuration{},
			"10s": NullDurationFrom(10 * time.Second),
		},
		{"Iterations", "K6_ITERATIONS"}: {
			"":    null.Int{},
			"123": null.IntFrom(123),
		},
		{"Stages", "K6_STAGES"}: {
			// "": []Stage{},
			"1s": []Stage{{
				Duration: NullDurationFrom(1 * time.Second)},
			},
			"1s:100": []Stage{
				{Duration: NullDurationFrom(1 * time.Second), Target: null.IntFrom(100)},
			},
			"1s,2s:100": []Stage{
				{Duration: NullDurationFrom(1 * time.Second)},
				{Duration: NullDurationFrom(2 * time.Second), Target: null.IntFrom(100)},
			},
		},
		{"MaxRedirects", "K6_MAX_REDIRECTS"}: {
			"":    null.Int{},
			"123": null.IntFrom(123),
		},
		{"InsecureSkipTLSVerify", "K6_INSECURE_SKIP_TLS_VERIFY"}: {
			"":      null.Bool{},
			"true":  null.BoolFrom(true),
			"false": null.BoolFrom(false),
		},
		// TLSCipherSuites
		// TLSVersion
		// TLSAuth
		{"NoConnectionReuse", "K6_NO_CONNECTION_REUSE"}: {
			"":      null.Bool{},
			"true":  null.BoolFrom(true),
			"false": null.BoolFrom(false),
		},
		{"UserAgent", "K6_USER_AGENT"}: {
			"":    null.String{},
			"Hi!": null.StringFrom("Hi!"),
		},
		{"Throw", "K6_THROW"}: {
			"":      null.Bool{},
			"true":  null.BoolFrom(true),
			"false": null.BoolFrom(false),
		},
		// Thresholds
		// External
	}
	for field, data := range testdata {
		os.Clearenv()
		t.Run(field.Name, func(t *testing.T) {
			for str, val := range data {
				t.Run(`"`+str+`"`, func(t *testing.T) {
					assert.NoError(t, os.Setenv(field.Key, str))
					var opts Options
					assert.NoError(t, envconfig.Process("k6", &opts))
					assert.Equal(t, val, reflect.ValueOf(opts).FieldByName(field.Name).Interface())
				})
			}
		})
	}
}
