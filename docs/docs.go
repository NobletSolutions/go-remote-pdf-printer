// Package docs Code generated by swaggo/swag. DO NOT EDIT
package docs

import "github.com/swaggo/swag"

const docTemplate = `{
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
        "/pdf": {
            "post": {
                "description": "Submit urls/data to be converted to a PDF",
                "consumes": [
                    "application/json",
                    "text/xml"
                ],
                "produces": [
                    "application/json"
                ],
                "summary": "Submit urls/data to be converted to a PDF",
                "parameters": [
                    {
                        "description": "The input todo struct",
                        "name": "data",
                        "in": "body",
                        "required": true,
                        "schema": {
                            "$ref": "#/definitions/main.PdfRequest"
                        }
                    }
                ],
                "responses": {
                    "200": {
                        "description": "OK",
                        "schema": {
                            "$ref": "#/definitions/main.PdfResponse"
                        }
                    },
                    "400": {
                        "description": "Bad Request"
                    },
                    "500": {
                        "description": "Internal Server Error"
                    }
                }
            }
        },
        "/png": {
            "post": {
                "description": "Submit a single url or data to be converted to a png",
                "consumes": [
                    "application/json",
                    "text/xml"
                ],
                "produces": [
                    "application/json"
                ],
                "summary": "Submit a single url or data to be converted to a png",
                "parameters": [
                    {
                        "description": "The input request",
                        "name": "data",
                        "in": "body",
                        "required": true,
                        "schema": {
                            "$ref": "#/definitions/main.PngRequest"
                        }
                    }
                ],
                "responses": {
                    "200": {
                        "description": "OK",
                        "schema": {
                            "$ref": "#/definitions/main.PngResponse"
                        }
                    },
                    "400": {
                        "description": "Bad Request"
                    },
                    "500": {
                        "description": "Internal Server Error"
                    }
                }
            }
        },
        "/preview": {
            "post": {
                "description": "Submit urls/data to be converted to a PDF and then one image per page",
                "consumes": [
                    "application/json",
                    "text/xml"
                ],
                "produces": [
                    "application/json"
                ],
                "summary": "Submit urls/data to be converted to a PDF and then one image per page",
                "parameters": [
                    {
                        "description": "The input todo struct",
                        "name": "data",
                        "in": "body",
                        "required": true,
                        "schema": {
                            "$ref": "#/definitions/main.PdfRequest"
                        }
                    }
                ],
                "responses": {
                    "200": {
                        "description": "OK",
                        "schema": {
                            "$ref": "#/definitions/main.PdfPreviewResponse"
                        }
                    },
                    "400": {
                        "description": "Bad Request"
                    },
                    "500": {
                        "description": "Internal Server Error"
                    }
                }
            }
        }
    },
    "definitions": {
        "main.PdfPreviewResponse": {
            "type": "object",
            "properties": {
                "images": {
                    "type": "array",
                    "items": {
                        "type": "string"
                    }
                },
                "pages": {
                    "type": "integer"
                }
            }
        },
        "main.PdfRequest": {
            "type": "object",
            "properties": {
                "data": {
                    "type": "array",
                    "items": {
                        "type": "string"
                    }
                },
                "download": {
                    "type": "boolean"
                },
                "footer": {
                    "type": "string"
                },
                "header": {
                    "type": "string"
                },
                "marginBottom": {
                    "type": "number"
                },
                "marginLeft": {
                    "type": "number"
                },
                "marginRight": {
                    "type": "number"
                },
                "marginTop": {
                    "type": "number"
                },
                "paperSize": {
                    "type": "array",
                    "items": {
                        "type": "number"
                    }
                }
            }
        },
        "main.PdfResponse": {
            "type": "object",
            "properties": {
                "components": {
                    "type": "array",
                    "items": {
                        "type": "string"
                    }
                },
                "url": {
                    "type": "string"
                }
            }
        },
        "main.PngRequest": {
            "type": "object",
            "properties": {
                "data": {
                    "type": "string"
                },
                "download": {
                    "type": "boolean"
                },
                "height": {
                    "type": "number"
                },
                "scale": {
                    "type": "number"
                },
                "width": {
                    "type": "number"
                },
                "x": {
                    "type": "number"
                },
                "y": {
                    "type": "number"
                }
            }
        },
        "main.PngResponse": {
            "type": "object",
            "properties": {
                "png": {
                    "type": "string"
                },
                "url": {
                    "type": "string"
                }
            }
        }
    }
}`

// SwaggerInfo holds exported Swagger Info so clients can modify it
var SwaggerInfo = &swag.Spec{
	Version:          "",
	Host:             "",
	BasePath:         "",
	Schemes:          []string{},
	Title:            "",
	Description:      "",
	InfoInstanceName: "swagger",
	SwaggerTemplate:  docTemplate,
	LeftDelim:        "{{",
	RightDelim:       "}}",
}

func init() {
	swag.Register(SwaggerInfo.InstanceName(), SwaggerInfo)
}
