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
        "termsOfService": "http://swagger.io/terms/",
        "contact": {
            "name": "API Support",
            "url": "http://www.swagger.io/support",
            "email": "support@swagger.io"
        },
        "license": {
            "name": "Apache 2.0",
            "url": "http://www.apache.org/licenses/LICENSE-2.0.html"
        },
        "version": "{{.Version}}"
    },
    "host": "{{.Host}}",
    "basePath": "{{.BasePath}}",
    "paths": {
        "/captcha": {
            "get": {
                "description": "请求base64编码的图像验证码",
                "tags": [
                    "共有路由"
                ],
                "responses": {
                    "200": {
                        "description": "{\"success\":true,\"id\":id,\"b64s\":base64编码的图像}",
                        "schema": {
                            "type": "string"
                        }
                    }
                }
            }
        },
        "/deleteUser": {
            "post": {
                "description": "获取用户信息",
                "tags": [
                    "私有路由"
                ],
                "parameters": [
                    {
                        "type": "string",
                        "description": "页面需要token鉴权，header带上Authorization字段",
                        "name": "data",
                        "in": "header",
                        "required": true
                    }
                ],
                "responses": {
                    "200": {
                        "description": "{\"success\":true,\"msg\":\"登录成功\",}",
                        "schema": {
                            "type": "string"
                        }
                    }
                }
            }
        },
        "/login": {
            "post": {
                "description": "提交登录信息",
                "tags": [
                    "共有路由"
                ],
                "parameters": [
                    {
                        "description": "上传登录信息和验证码",
                        "name": "data",
                        "in": "body",
                        "required": true,
                        "schema": {
                            "$ref": "#/definitions/sysRequest.Login"
                        }
                    }
                ],
                "responses": {
                    "200": {
                        "description": "{\"success\":true,\"msg\":\"登录成功\",\"token\":\"aaa.bbb.ccc\"}",
                        "schema": {
                            "type": "string"
                        }
                    }
                }
            }
        },
        "/ping": {
            "get": {
                "description": "get string by ID",
                "produces": [
                    "application/json"
                ],
                "tags": [
                    "Example API"
                ],
                "summary": "Show a account",
                "responses": {
                    "200": {
                        "description": ""
                    }
                }
            }
        },
        "/register": {
            "post": {
                "description": "提交注册用户信息",
                "produces": [
                    "application/json"
                ],
                "tags": [
                    "共有路由"
                ],
                "parameters": [
                    {
                        "description": "注册用户账户,密码",
                        "name": "data",
                        "in": "body",
                        "required": true,
                        "schema": {
                            "$ref": "#/definitions/sysRequest.Register"
                        }
                    }
                ],
                "responses": {
                    "200": {
                        "description": "{\"success\":true,\"msg\":\"注册成功\"}",
                        "schema": {
                            "type": "string"
                        }
                    }
                }
            }
        },
        "/userinfo": {
            "get": {
                "description": "获取用户信息",
                "tags": [
                    "私有路由"
                ],
                "parameters": [
                    {
                        "type": "string",
                        "description": "页面需要token鉴权，header带上Authorization字段",
                        "name": "data",
                        "in": "header",
                        "required": true
                    }
                ],
                "responses": {
                    "200": {
                        "description": "{\"success\":true,\"msg\":\"hello:user\",}",
                        "schema": {
                            "type": "string"
                        }
                    }
                }
            }
        }
    },
    "definitions": {
        "sysRequest.Login": {
            "type": "object",
            "properties": {
                "b64s": {
                    "type": "string"
                },
                "id": {
                    "type": "string"
                },
                "password": {
                    "type": "string"
                },
                "username": {
                    "type": "string"
                }
            }
        },
        "sysRequest.Register": {
            "type": "object",
            "properties": {
                "password": {
                    "type": "string"
                },
                "username": {
                    "type": "string"
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
	Version:     "",
	Host:        "",
	BasePath:    "",
	Schemes:     []string{},
	Title:       "",
	Description: "",
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
