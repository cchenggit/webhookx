package tracing

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/go-resty/resty/v2"
	. "github.com/onsi/ginkgo/v2"

	"github.com/stretchr/testify/assert"
	"github.com/webhookx-io/webhookx/admin/api"
	"github.com/webhookx-io/webhookx/app"
	"github.com/webhookx-io/webhookx/db/entities"
	"github.com/webhookx-io/webhookx/test/helper"
	"github.com/webhookx-io/webhookx/utils"
)

var _ = Describe("tracing admin", Ordered, func() {
	endpoints := map[string]string{
		"http": "http://localhost:4318/v1/traces",
		"grpc": "localhost:4317",
	}
	for protocol, address := range endpoints {
		Context(protocol, func() {
			var app *app.Application
			var proxyClient *resty.Client
			var adminClient *resty.Client
			entitiesConfig := helper.EntitiesConfig{
				Endpoints: []*entities.Endpoint{helper.DefaultEndpoint()},
				Sources:   []*entities.Source{helper.DefaultSource()},
			}
			entitiesConfig.Sources[0].Async = true

			BeforeAll(func() {
				helper.InitOtelOutput()
				helper.InitDB(true, &entitiesConfig)
				proxyClient = helper.ProxyClient()
				adminClient = helper.AdminClient()

				envs := map[string]string{
					"WEBHOOKX_ADMIN_LISTEN":                    "0.0.0.0:8080",
					"WEBHOOKX_PROXY_LISTEN":                    "0.0.0.0:8081",
					"WEBHOOKX_TRACING_SERVICENAME":             "WebhookX", // env splite by _
					"WEBHOOKX_TRACING_CAPTUREDREQUESTHEADERS":  "X-Request-Id,Content-Type,Accept",
					"WEBHOOKX_TRACING_CAPTUREDRESPONSEHEADERS": "Content-Type",
					"WEBHOOKX_TRACING_SAFEQUERYPARAMS":         "page_no",
					"WEBHOOKX_TRACING_SAMPLINGRATE":            "1",
				}

				if protocol == "http" {
					envs["WEBHOOKX_TRACING_OPENTELEMETRY_HTTP_ENDPOINT"] = address
				} else {
					envs["WEBHOOKX_TRACING_OPENTELEMETRY_GRPC_ENDPOINT"] = address
				}
				app = utils.Must(helper.Start(envs))
			})

			AfterAll(func() {
				app.Stop()
			})

			It("sanity", func() {
				expectedScopeName := "github.com/webhookx-io/webhookx"
				entrypoint := map[string]string{
					"entrypoint":                        "admin",
					"http.request.method":               "GET",
					"network.protocol.version":          "*",
					"http.request.body.size":            "0",
					"url.path":                          "/workspaces/default/attempts",
					"url.query":                         "page_no=1",
					"url.scheme":                        "*",
					"user_agent.original":               "*",
					"server.address":                    "localhost:8080",
					"network.peer.address":              "*",
					"client.address":                    "*",
					"client.port":                       "*",
					"network.peer.port":                 "*",
					"http.response.status_code":         "200",
					"http.response.header.content-type": "application/json",
					"http.request.header.x-request-id":  "111111111",
				}

				expectedScopeSpans := map[string]map[string]string{
					"entrypoint": entrypoint,
					"dao.page":   {},
					"dao.count":  {},
					"dao.list":   {},
				}
				// wait for export
				proxyFunc := func() bool {
					resp, err := proxyClient.R().
						SetBody(`{
							"event_type": "foo.bar",
							"data": {
								"key": "value"
							}
						}`).Post("/")
					return err == nil && resp.StatusCode() == 200
				}
				assert.Eventually(GinkgoT(), proxyFunc, time.Second*5, time.Second)

				n, err := helper.FileCountLine(helper.OtelCollectorTracesFile)
				assert.Nil(GinkgoT(), err)
				n++

				// make more tracing data
				for i := 0; i < 20; i++ {
					go proxyFunc()
				}
				assert.Eventually(GinkgoT(), func() bool {
					resp, err := adminClient.R().
						SetHeaders(map[string]string{
							"X-Request-Id": "111111111",
						}).
						SetResult(api.Pagination[*entities.Attempt]{}).
						Get("/workspaces/default/attempts?page_no=1")
					result := resp.Result().(*api.Pagination[*entities.Attempt])
					return err == nil && resp.StatusCode() == 200 && len(result.Data) == 20
				}, time.Second*10, time.Second)

				gotScopeNames := make(map[string]bool)
				gotSpanAttributes := make(map[string]map[string]string)
				assert.Eventually(GinkgoT(), func() bool {
					line, err := helper.FileLine(helper.OtelCollectorTracesFile, n)
					if err != nil || line == "" {
						return false
					}
					n++
					var trace ExportedTrace
					err = json.Unmarshal([]byte(line), &trace)
					if err != nil {
						return false
					}

					if len(trace.ResourceSpans) == 0 {
						return false
					}
					trace.mergeTo(gotScopeNames, gotSpanAttributes)

					if !gotScopeNames[expectedScopeName] {
						fmt.Printf("scope %s not exist", expectedScopeName)
						fmt.Println("")
						return false
					}

					for spanName, expectedAttributes := range expectedScopeSpans {
						gotAttributes, ok := gotSpanAttributes[spanName]
						if !ok {
							fmt.Printf("span %s not exist", spanName)
							fmt.Println()
							return false
						}

						if len(expectedAttributes) > 0 {
							for k, v := range expectedAttributes {
								if _, ok := gotAttributes[k]; !ok {
									fmt.Printf("expected span %s attribute %s not exist", spanName, k)
									fmt.Println("")
									return false
								}
								valMatch := (v == "*" || gotAttributes[k] == v)
								if !valMatch {
									fmt.Printf("expected span %s attribute %s value not match: %s", spanName, k, v)
									fmt.Println("")
									return false
								}
							}
						}
					}
					return true
				}, time.Second*30, time.Second)
			})
		})
	}

})
