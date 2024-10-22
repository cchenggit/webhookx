package tracing

type ExportedTrace struct {
	ResourceSpans []struct {
		Resource struct {
			Attributes []struct {
				Key   string `json:"key"`
				Value struct {
					ArrayValue *struct {
						Values []struct {
							StringValue string `json:"stringValue"`
						} `json:"values"`
					} `json:"arrayValue,omitempty"`
					IntValue    *string `json:"intValue,omitempty"`
					StringValue *string `json:"stringValue,omitempty"`
				} `json:"value"`
			} `json:"attributes"`
		} `json:"resource"`
		SchemaURL  string `json:"schemaUrl"`
		ScopeSpans []struct {
			Scope struct {
				Name    string  `json:"name"`
				Version *string `json:"version,omitempty"`
			} `json:"scope"`
			Spans []struct {
				Attributes []struct {
					Key   string `json:"key"`
					Value struct {
						IntValue    *string `json:"intValue,omitempty"`
						StringValue *string `json:"stringValue,omitempty"`
						ArrayValue  *struct {
							Values []struct {
								StringValue string `json:"stringValue"`
							} `json:"values"`
						} `json:"arrayValue,omitempty"`
					} `json:"value"`
				} `json:"attributes,omitempty"`
				EndTimeUnixNano   string `json:"endTimeUnixNano"`
				Flags             int    `json:"flags"`
				Kind              int    `json:"kind"`
				Name              string `json:"name"`
				ParentSpanID      string `json:"parentSpanId"`
				SpanID            string `json:"spanId"`
				StartTimeUnixNano string `json:"startTimeUnixNano"`
				Status            struct {
				} `json:"status"`
				TraceID string `json:"traceId"`
			} `json:"spans"`
		} `json:"scopeSpans"`
	} `json:"resourceSpans"`
}
