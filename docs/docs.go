// Package docs GENERATED BY THE COMMAND ABOVE; DO NOT EDIT
// This file was generated by swaggo/swag
package docs

import (
	"bytes"
	"encoding/json"
	"strings"
	"text/template"

	"github.com/swaggo/swag"
)

var doc = `{
    "schemes": {{ marshal .Schemes }},
    "swagger": "2.0",
    "info": {
        "description": "{{escape .Description}}",
        "title": "{{.Title}}",
        "contact": {},
        "version": "{{.Version}}"
    },
    "host": "{{.Host}}",
    "basePath": "{{.BasePath}}",
    "paths": {
        "/countries": {
            "get": {
                "description": "List number of providers in each country",
                "consumes": [
                    "application/json"
                ],
                "tags": [
                    "countries"
                ],
                "summary": "List number of providers in each country",
                "parameters": [
                    {
                        "type": "string",
                        "description": "Consumer country",
                        "name": "from",
                        "in": "query"
                    },
                    {
                        "type": "string",
                        "description": "Provider ID",
                        "name": "provider_id",
                        "in": "query"
                    },
                    {
                        "type": "string",
                        "description": "Service type",
                        "name": "service_type",
                        "in": "query"
                    },
                    {
                        "type": "string",
                        "description": "Provider country",
                        "name": "location_country",
                        "in": "query"
                    },
                    {
                        "type": "string",
                        "description": "IP type (residential, datacenter, etc.)",
                        "name": "ip_type",
                        "in": "query"
                    },
                    {
                        "type": "string",
                        "description": "Access policy. When empty, returns only public proposals (default). Use 'all' to return all.",
                        "name": "access_policy",
                        "in": "query"
                    },
                    {
                        "type": "string",
                        "description": "Access policy source",
                        "name": "access_policy_source",
                        "in": "query"
                    },
                    {
                        "type": "number",
                        "description": "Minimum compatibility. When empty, will not filter by it.",
                        "name": "compatibility_min",
                        "in": "query"
                    },
                    {
                        "type": "number",
                        "description": "Maximum compatibility. When empty, will not filter by it.",
                        "name": "compatibility_max",
                        "in": "query"
                    },
                    {
                        "type": "number",
                        "description": "Minimal quality threshold. When empty will be defaulted to 0. Quality ranges from [0.0; 3.0]",
                        "name": "quality_min",
                        "in": "query"
                    }
                ],
                "responses": {}
            }
        },
        "/ping": {
            "get": {
                "description": "Ping",
                "consumes": [
                    "application/json"
                ],
                "produces": [
                    "application/json"
                ],
                "tags": [
                    "system"
                ],
                "summary": "Ping",
                "responses": {
                    "200": {
                        "description": "OK",
                        "schema": {
                            "$ref": "#/definitions/health.PingResponse"
                        }
                    }
                }
            }
        },
        "/proposals": {
            "get": {
                "description": "List proposals",
                "consumes": [
                    "application/json"
                ],
                "tags": [
                    "proposals"
                ],
                "summary": "List proposals",
                "parameters": [
                    {
                        "type": "string",
                        "description": "Consumer country",
                        "name": "from",
                        "in": "query"
                    },
                    {
                        "type": "string",
                        "description": "Provider ID",
                        "name": "provider_id",
                        "in": "query"
                    },
                    {
                        "type": "string",
                        "description": "Service type",
                        "name": "service_type",
                        "in": "query"
                    },
                    {
                        "type": "string",
                        "description": "Provider country",
                        "name": "location_country",
                        "in": "query"
                    },
                    {
                        "type": "string",
                        "description": "IP type (residential, datacenter, etc.)",
                        "name": "ip_type",
                        "in": "query"
                    },
                    {
                        "type": "string",
                        "description": "Access policy. When empty, returns only public proposals (default). Use 'all' to return all.",
                        "name": "access_policy",
                        "in": "query"
                    },
                    {
                        "type": "string",
                        "description": "Access policy source",
                        "name": "access_policy_source",
                        "in": "query"
                    },
                    {
                        "type": "number",
                        "description": "Minimum compatibility. When empty, will not filter by it.",
                        "name": "compatibility_min",
                        "in": "query"
                    },
                    {
                        "type": "number",
                        "description": "Maximum compatibility. When empty, will not filter by it.",
                        "name": "compatibility_max",
                        "in": "query"
                    },
                    {
                        "type": "number",
                        "description": "Minimal quality threshold. When empty will be defaulted to 0. Quality ranges from [0.0; 3.0]",
                        "name": "quality_min",
                        "in": "query"
                    }
                ],
                "responses": {
                    "200": {
                        "description": "OK",
                        "schema": {
                            "type": "array",
                            "items": {
                                "$ref": "#/definitions/v3.Proposal"
                            }
                        }
                    }
                }
            }
        },
        "/proposals-metadata": {
            "get": {
                "description": "List proposals' metadata",
                "consumes": [
                    "application/json"
                ],
                "summary": "List proposals' metadata.",
                "parameters": [
                    {
                        "type": "string",
                        "description": "Provider ID",
                        "name": "provider_id",
                        "in": "query"
                    }
                ],
                "responses": {
                    "200": {
                        "description": "OK",
                        "schema": {
                            "type": "array",
                            "items": {
                                "$ref": "#/definitions/v3.Metadata"
                            }
                        }
                    }
                }
            }
        },
        "/status": {
            "get": {
                "description": "Status",
                "consumes": [
                    "application/json"
                ],
                "produces": [
                    "application/json"
                ],
                "tags": [
                    "system"
                ],
                "summary": "Status",
                "responses": {
                    "200": {
                        "description": "OK",
                        "schema": {
                            "$ref": "#/definitions/health.StatusResponse"
                        }
                    }
                }
            }
        }
    },
    "definitions": {
        "health.PingResponse": {
            "type": "object",
            "properties": {
                "message": {
                    "type": "string"
                }
            }
        },
        "health.StatusResponse": {
            "type": "object",
            "properties": {
                "cache_ok": {
                    "type": "boolean"
                }
            }
        },
        "v3.AccessPolicy": {
            "type": "object",
            "properties": {
                "id": {
                    "type": "string"
                },
                "source": {
                    "type": "string"
                }
            }
        },
        "v3.Contact": {
            "type": "object",
            "properties": {
                "definition": {
                    "type": "object"
                },
                "type": {
                    "type": "string"
                }
            }
        },
        "v3.Location": {
            "type": "object",
            "properties": {
                "asn": {
                    "type": "integer"
                },
                "city": {
                    "type": "string"
                },
                "continent": {
                    "type": "string"
                },
                "country": {
                    "type": "string"
                },
                "ip_type": {
                    "type": "string"
                },
                "isp": {
                    "type": "string"
                },
                "region": {
                    "type": "string"
                }
            }
        },
        "v3.Metadata": {
            "type": "object",
            "properties": {
                "country": {
                    "type": "string"
                },
                "ip_type": {
                    "type": "string"
                },
                "isp": {
                    "type": "string"
                },
                "monitoring_failed": {
                    "type": "boolean"
                },
                "provider_id": {
                    "type": "string"
                },
                "service_type": {
                    "type": "string"
                },
                "updated_at": {
                    "type": "string"
                },
                "whitelist": {
                    "type": "boolean"
                }
            }
        },
        "v3.Proposal": {
            "type": "object",
            "properties": {
                "access_policies": {
                    "type": "array",
                    "items": {
                        "$ref": "#/definitions/v3.AccessPolicy"
                    }
                },
                "compatibility": {
                    "type": "integer"
                },
                "contacts": {
                    "type": "array",
                    "items": {
                        "$ref": "#/definitions/v3.Contact"
                    }
                },
                "format": {
                    "type": "string"
                },
                "id": {
                    "type": "integer"
                },
                "location": {
                    "$ref": "#/definitions/v3.Location"
                },
                "provider_id": {
                    "type": "string"
                },
                "quality": {
                    "$ref": "#/definitions/v3.Quality"
                },
                "service_type": {
                    "type": "string"
                },
                "tags": {
                    "type": "array",
                    "items": {
                        "type": "string"
                    }
                }
            }
        },
        "v3.Quality": {
            "type": "object",
            "properties": {
                "bandwidth": {
                    "description": "Bandwidth in Mbps.",
                    "type": "number"
                },
                "latency": {
                    "description": "Latency in ms.",
                    "type": "number"
                },
                "monitoring_failed": {
                    "description": "MonitoringFailed did monitoring agent succeed to connect to the node.",
                    "type": "boolean"
                },
                "quality": {
                    "description": "Quality valuation from the oracle.",
                    "type": "number"
                },
                "uptime": {
                    "description": "Uptime in hours per day",
                    "type": "number"
                }
            }
        }
    }
}`

type swaggerInfo struct {
	Version     string
	Host        string
	BasePath    string
	Schemes     []string
	Title       string
	Description string
}

// SwaggerInfo holds exported Swagger Info so clients can modify it
var SwaggerInfo = swaggerInfo{
	Version:     "3.0",
	Host:        "",
	BasePath:    "/api/v3",
	Schemes:     []string{},
	Title:       "Discovery API",
	Description: "Discovery API for Mysterium Network",
}

type s struct{}

func (s *s) ReadDoc() string {
	sInfo := SwaggerInfo
	sInfo.Description = strings.Replace(sInfo.Description, "\n", "\\n", -1)

	t, err := template.New("swagger_info").Funcs(template.FuncMap{
		"marshal": func(v interface{}) string {
			a, _ := json.Marshal(v)
			return string(a)
		},
		"escape": func(v interface{}) string {
			// escape tabs
			str := strings.Replace(v.(string), "\t", "\\t", -1)
			// replace " with \", and if that results in \\", replace that with \\\"
			str = strings.Replace(str, "\"", "\\\"", -1)
			return strings.Replace(str, "\\\\\"", "\\\\\\\"", -1)
		},
	}).Parse(doc)
	if err != nil {
		return doc
	}

	var tpl bytes.Buffer
	if err := t.Execute(&tpl, sInfo); err != nil {
		return doc
	}

	return tpl.String()
}

func init() {
	swag.Register("swagger", &s{})
}
