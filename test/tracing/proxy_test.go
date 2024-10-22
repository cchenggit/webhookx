package tracing

import (
	"encoding/json"
	"fmt"
	"time"

	"testing"

	"github.com/go-resty/resty/v2"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/stretchr/testify/assert"
	"github.com/webhookx-io/webhookx/app"
	"github.com/webhookx-io/webhookx/db/entities"
	"github.com/webhookx-io/webhookx/test/helper"
	"github.com/webhookx-io/webhookx/utils"
)

func TestTracing(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Tracing Suite")
}

var _ = Describe("tracing proxy", Ordered, func() {
	endpoints := map[string]string{
		"http": "http://localhost:4318/v1/traces",
		"grpc": "localhost:4317",
	}
	for protocol, address := range endpoints {
		Context(protocol, func() {
			var app *app.Application
			var proxyClient *resty.Client

			entitiesConfig := helper.EntitiesConfig{
				Endpoints: []*entities.Endpoint{helper.DefaultEndpoint()},
				Sources:   []*entities.Source{helper.DefaultSource()},
			}
			entitiesConfig.Sources[0].Async = false

			BeforeAll(func() {
				helper.InitOtelOutput()
				helper.InitDB(true, &entitiesConfig)
				proxyClient = helper.ProxyClient()

				envs := map[string]string{
					"WEBHOOKX_PROXY_LISTEN":                    "0.0.0.0:8081",
					"WEBHOOKX_TRACING_SERVICENAME":             "WebhookX", // env splite by _
					"WEBHOOKX_TRACING_CAPTUREDREQUESTHEADERS":  "X-Request-Id,Content-Type,Accept",
					"WEBHOOKX_TRACING_CAPTUREDRESPONSEHEADERS": "Content-Type",
					"WEBHOOKX_TRACING_SAFEQUERYPARAMS":         "test",
					"WEBHOOKX_TRACING_SAMPLINGRATE":            "1",
					"WEBHOOKX_TRACING_GLOBALATTRIBUTES":        "env:dev",
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
				helper.TruncateFile(helper.OtelCollectorTracesFile)
			})

			It("sanity", func() {
				expectedScopeName := "github.com/webhookx-io/webhookx"
				entrypoint := map[string]string{
					"entrypoint":                       "proxy",
					"http.request.method":              "POST",
					"network.protocol.version":         "*",
					"http.request.body.size":           "*",
					"url.path":                         "/",
					"url.query":                        "test=true",
					"url.scheme":                       "*",
					"user_agent.original":              "*",
					"server.address":                   "*",
					"network.peer.address":             "*",
					"client.address":                   "*",
					"client.port":                      "*",
					"network.peer.port":                "*",
					"http.response.status_code":        "200",
					"http.request.header.x-request-id": "123456789",
					"http.request.header.content-type": "application/json",
					"http.request.header.accept":       "application/json",
				}
				router := map[string]string{
					"router.id":          "*",
					"router.name":        "*",
					"router.workspaceId": "*",
					"http.route":         "/",
				}
				expectedScopeSpans := map[string]map[string]string{
					"entrypoint":       entrypoint,
					"router":           router,
					"dispatcher":       {},
					"dao.insert":       {},
					"db.transaction":   {},
					"dao.batch_insert": {},
					"dao.list":         {},
				}
				// wait for export
				proxyFunc := func() bool {
					resp, err := proxyClient.R().
						SetBody(`{
					"event_type": "foo.bar",
					"data": {
						"key": "value"
					}
				}`).
						SetHeaders(map[string]string{
							"Content-Type": "application/json",
							"Accept":       "application/json",
							"X-Request-Id": "123456789",
						}).
						SetQueryParam("test", "true").
						Post("/")
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

					for _, resourceSpan := range trace.ResourceSpans {
						scopeSpans := resourceSpan.ScopeSpans
						for _, scopeSpan := range scopeSpans {
							gotScopeNames[scopeSpan.Scope.Name] = true
							for _, span := range scopeSpan.Spans {
								attributes := make(map[string]string)
								for _, attr := range span.Attributes {
									if attr.Value.StringValue != nil {
										attributes[attr.Key] = *attr.Value.StringValue
									} else if attr.Value.IntValue != nil {
										attributes[attr.Key] = *attr.Value.IntValue
									} else if attr.Value.ArrayValue != nil {
										if len(attr.Value.ArrayValue.Values) == 1 {
											attributes[attr.Key] = attr.Value.ArrayValue.Values[0].StringValue
										} else {
											var values []string
											for _, v := range attr.Value.ArrayValue.Values {
												values = append(values, v.StringValue)
											}
											attributes[attr.Key] = fmt.Sprintf("[%s]", values)
										}
									}
								}
								gotSpanAttributes[span.Name] = attributes
							}
						}
					}

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

	Context("SDK configuration by env", func() {
		var app *app.Application
		var proxyClient *resty.Client

		entitiesConfig := helper.EntitiesConfig{
			Endpoints: []*entities.Endpoint{helper.DefaultEndpoint()},
			Sources:   []*entities.Source{helper.DefaultSource()},
		}
		entitiesConfig.Sources[0].Async = false

		BeforeAll(func() {
			var err error
			helper.InitOtelOutput()
			helper.InitDB(true, &entitiesConfig)
			proxyClient = helper.ProxyClient()

			app, err = helper.Start(map[string]string{
				"WEBHOOKX_PROXY_LISTEN":                        "0.0.0.0:8081",
				"WEBHOOKX_TRACING_SERVICENAME":                 "WebhookX", // env splite by _
				"WEBHOOKX_TRACING_CAPTUREDREQUESTHEADERS":      "X-Request-Id,Content-Type,Accept",
				"WEBHOOKX_TRACING_CAPTUREDRESPONSEHEADERS":     "Content-Type",
				"WEBHOOKX_TRACING_SAFEQUERYPARAMS":             "test",
				"WEBHOOKX_TRACING_SAMPLINGRATE":                "1",
				"WEBHOOKX_TRACING_GLOBALATTRIBUTES":            "env:test",
				"WEBHOOKX_TRACING_OPENTELEMETRY_HTTP_ENDPOINT": "http://localhost:4318/v1/traces",
				"OTEL_RESOURCE_ATTRIBUTES":                     "service.version=0.3",
				"OTEL_SERVICE_NAME":                            "WebhookX-Test", // env override
			})
			assert.Nil(GinkgoT(), err)
		})

		AfterAll(func() {
			app.Stop()
			helper.TruncateFile(helper.OtelCollectorTracesFile)
		})

		It("sanity", func() {
			n, err := helper.FileCountLine(helper.OtelCollectorTracesFile)
			assert.Nil(GinkgoT(), err)
			n++
			assert.Eventually(GinkgoT(), func() bool {
				resp, err := proxyClient.R().
					SetBody(`{
					"event_type": "foo.bar",
					"data": {
						"key": "value"
					}
				}`).
					SetHeaders(map[string]string{
						"Content-Type": "application/json",
						"Accept":       "application/json",
						"X-Request-Id": "123456789",
					}).
					SetQueryParam("test", "true").
					Post("/")
				return err == nil && resp.StatusCode() == 200
			}, time.Second*5, time.Second)

			expected := map[string]string{"service.name": "WebhookX-Test", "service.version": "0.3", "env": "test"}
			assert.Eventually(GinkgoT(), func() bool {
				line, err := helper.FileLine(helper.OtelCollectorTracesFile, n)
				if err != nil || line == "" {
					return false
				}
				n++
				var req ExportedTrace
				_ = json.Unmarshal([]byte(line), &req)
				attributesMap := make(map[string]string)
				for _, resourceSpan := range req.ResourceSpans {
					for _, attr := range resourceSpan.Resource.Attributes {
						if attr.Value.StringValue != nil {
							attributesMap[attr.Key] = *attr.Value.StringValue
						}
					}
				}
				for name, expectVal := range expected {
					if val, ok := attributesMap[name]; !ok || val != expectVal {
						fmt.Printf("expected attribute %s not exist or value %s not match", name, val)
						fmt.Println("")
						return false
					}
				}
				return true
			}, time.Second*30, time.Second)
		})
	})
})
