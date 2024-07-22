package envoyconfig

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/encoding/protojson"

	"github.com/pomerium/pomerium/config"
	"github.com/pomerium/pomerium/config/envoyconfig/filemgr"
	"github.com/pomerium/pomerium/internal/testutil"
	"github.com/pomerium/pomerium/pkg/cryptutil"
)

func TestBuilder_buildMainRouteConfiguration(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	cfg := &config.Config{Options: &config.Options{
		CookieName:             "pomerium",
		DefaultUpstreamTimeout: time.Second * 3,
		SharedKey:              cryptutil.NewBase64Key(),
		Services:               "proxy",
		Policies: []config.Policy{
			{
				From: "https://*.example.com",
				To:   mustParseWeightedURLs(t, "https://www.example.com"),
			},
		},
	}}
	b := New("grpc", "http", "metrics", filemgr.NewManager(), nil)
	sb := b.WithConfig(cfg)
	routeConfiguration, err := sb.buildMainRouteConfiguration(ctx)
	assert.NoError(t, err)
	testutil.AssertProtoJSONEqual(t, `{
		"name": "main",
		"validateClusters": false,
		"virtualHosts": [
			{
				"name": "catch-all",
				"domains": ["*"],
				"routes": [
					`+protojson.Format(sb.buildControlPlanePathRoute(context.Background(), "/ping"))+`,
					`+protojson.Format(sb.buildControlPlanePathRoute(context.Background(), "/healthz"))+`,
					`+protojson.Format(sb.buildControlPlanePathRoute(context.Background(), "/.pomerium"))+`,
					`+protojson.Format(sb.buildControlPlanePrefixRoute(context.Background(), "/.pomerium/"))+`,
					`+protojson.Format(sb.buildControlPlanePathRoute(context.Background(), "/.well-known/pomerium"))+`,
					`+protojson.Format(sb.buildControlPlanePrefixRoute(context.Background(), "/.well-known/pomerium/"))+`,
					{
						"name": "policy-0",
						"match": {
							"headers": [
								{ "name": ":authority", "stringMatch": { "safeRegex": { "regex": "^(.*)\\.example\\.com$" } }}
							],
							"prefix": "/"
						},
						"metadata": {
							"filterMetadata": {
								"envoy.filters.http.lua": {
									"remove_impersonate_headers": false,
									"remove_pomerium_authorization": true,
									"remove_pomerium_cookie": "pomerium",
									"rewrite_response_headers": []
								}
							}
						},
						"requestHeadersToRemove": [
							"x-pomerium-jwt-assertion",
							"x-pomerium-jwt-assertion-for",
							"x-pomerium-reproxy-policy",
							"x-pomerium-reproxy-policy-hmac"
						],
						"responseHeadersToAdd": [
							{ "appendAction": "OVERWRITE_IF_EXISTS_OR_ADD", "header": { "key": "X-Frame-Options", "value": "SAMEORIGIN" } },
							{ "appendAction": "OVERWRITE_IF_EXISTS_OR_ADD", "header": { "key": "X-XSS-Protection", "value": "1; mode=block" } }
						],
						"route": {
							"autoHostRewrite": true,
							"cluster": "route-f6a1c77f275e05b4",
							"hashPolicy": [
								{ "header": { "headerName": "x-pomerium-routing-key" }, "terminal": true },
								{ "connectionProperties": { "sourceIp": true }, "terminal": true }
							],
							"timeout": "3s",
							"upgradeConfigs": [
								{ "enabled": false, "upgradeType": "websocket" },
								{ "enabled": false, "upgradeType": "spdy/3.1" }
							]
						},
						"typedPerFilterConfig": {
							"envoy.filters.http.ext_authz": {
								"@type": "type.googleapis.com/envoy.extensions.filters.http.ext_authz.v3.ExtAuthzPerRoute",
								"checkSettings": {
									"contextExtensions": {
										"internal": "false",
										"route_id": "17771704953515935156"
									}
								}
							}
						}
					},
					{
						"name": "policy-0",
						"match": {
							"headers": [
								{ "name": ":authority", "stringMatch": { "safeRegex": { "regex": "^(.*)\\.example\\.com:443$" } }}
							],
							"prefix": "/"
						},
						"metadata": {
							"filterMetadata": {
								"envoy.filters.http.lua": {
									"remove_impersonate_headers": false,
									"remove_pomerium_authorization": true,
									"remove_pomerium_cookie": "pomerium",
									"rewrite_response_headers": []
								}
							}
						},
						"requestHeadersToRemove": [
							"x-pomerium-jwt-assertion",
							"x-pomerium-jwt-assertion-for",
							"x-pomerium-reproxy-policy",
							"x-pomerium-reproxy-policy-hmac"
						],
						"responseHeadersToAdd": [
							{ "appendAction": "OVERWRITE_IF_EXISTS_OR_ADD", "header": { "key": "X-Frame-Options", "value": "SAMEORIGIN" } },
							{ "appendAction": "OVERWRITE_IF_EXISTS_OR_ADD", "header": { "key": "X-XSS-Protection", "value": "1; mode=block" } }
						],
						"route": {
							"autoHostRewrite": true,
							"cluster": "route-f6a1c77f275e05b4",
							"hashPolicy": [
								{ "header": { "headerName": "x-pomerium-routing-key" }, "terminal": true },
								{ "connectionProperties": { "sourceIp": true }, "terminal": true }
							],
							"timeout": "3s",
							"upgradeConfigs": [
								{ "enabled": false, "upgradeType": "websocket" },
								{ "enabled": false, "upgradeType": "spdy/3.1" }
							]
						},
						"typedPerFilterConfig": {
							"envoy.filters.http.ext_authz": {
								"@type": "type.googleapis.com/envoy.extensions.filters.http.ext_authz.v3.ExtAuthzPerRoute",
								"checkSettings": {
									"contextExtensions": {
										"internal": "false",
										"route_id": "17771704953515935156"
									}
								}
							}
						}
					}
				]
			}
		]

	}`, routeConfiguration)
}
