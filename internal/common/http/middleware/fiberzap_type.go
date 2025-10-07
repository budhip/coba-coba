package middleware

import (
	"bytes"
	"encoding/json"
	"strings"

	"bitbucket.org/Amartha/go-fp-transaction/internal/common"

	xlog "bitbucket.org/Amartha/go-x/log"
	"github.com/gofiber/fiber/v2"
	"github.com/valyala/fasthttp"
)

type req struct {
	body     string
	fullPath string
	user     string
	ip       string
	method   string
	route    string
	headers  *headerbag
}

// use struct instead bool because empty struct is zero size, while bool is 1 byte
var sensitiveHeaders = map[string]struct{}{
	"authorization": {},
	"cookie":        {},
	"set-cookie":    {},
	"x-secret-key":  {},
	// Add other sensitive headers here
}

func Req(c *fiber.Ctx) *req {
	reqq := c.Request()
	var body []byte
	buffer := new(bytes.Buffer)
	err := json.Compact(buffer, reqq.Body())
	if err != nil {
		body = reqq.Body()
	} else {
		body = buffer.Bytes()
	}

	headers := &headerbag{
		vals: make(map[string]string),
	}
	reqq.Header.VisitAll(func(key, val []byte) {
		k := bytes.NewBuffer(key).String()
		v := bytes.NewBuffer(val).String()

		lowerKey := strings.ToLower(k)
		if _, exists := sensitiveHeaders[lowerKey]; exists || strings.Contains(lowerKey, "secret") {
			// Hide the value
			v = "*****"
		}

		headers.vals[lowerKey] = v
	})

	var userEmail string
	if u := c.Locals("userEmail"); u != nil {
		userEmail = u.(string)
	}

	return &req{
		body:     bytes.NewBuffer(body).String(),
		fullPath: bytes.NewBuffer(reqq.RequestURI()).String(),
		headers:  headers,
		ip:       c.IP(),
		method:   c.Method(),
		route:    c.Route().Path,
		user:     userEmail,
	}
}

func (r *req) MarshalLogObject(enc xlog.ObjectEncoder) error {
	enc.AddString("ip", r.ip)
	enc.AddString("fullPath", r.fullPath)
	enc.AddString("method", r.method)
	enc.AddString("route", r.route)

	err := enc.AddObject("headers", r.headers)
	if err != nil {
		return err
	}

	if r.body != "" {
		enc.AddString("body", r.body)
	}

	if r.user != "" {
		enc.AddString("user", r.user)
	}

	return nil
}

type headerbag struct {
	vals map[string]string
}

func (h *headerbag) MarshalLogObject(enc xlog.ObjectEncoder) error {
	for k, v := range h.vals {
		enc.AddString(k, v)
	}

	return nil
}

type resp struct {
	code    int
	_type   string
	Code    int    `json:"code,omitempty"`
	Status  string `json:"status,omitempty"`
	Message string `json:"message,omitempty"`
	Data    string `json:"data,omitempty"`
	List    string `json:"list,omitempty"`
	Meta    string `json:"meta,omitempty"`
	Error   string `json:"error,omitempty"`
}

func Resp(r *fasthttp.Response) *resp {
	var res resBody
	json.Unmarshal(r.Body(), &res)

	bData, _ := json.Marshal(res.Data)
	bList, _ := json.Marshal(res.List)
	bMeta, _ := json.Marshal(res.Meta)
	bErr, _ := json.Marshal(res.Error)

	return &resp{
		code:    r.StatusCode(),
		_type:   bytes.NewBuffer(r.Header.ContentType()).String(),
		Code:    res.Code,
		Status:  res.Status,
		Message: res.Message,
		Data:    string(bData),
		List:    string(bList),
		Meta:    string(bMeta),
		Error:   string(bErr),
	}
}

func (r *resp) MarshalLogObject(enc xlog.ObjectEncoder) error {
	enc.AddString("type", r._type)
	enc.AddInt("code", r.code)
	enc.AddString("status", r.Status)

	if r.Message != "" {
		enc.AddString("message", r.Message)
	}
	if r.Data != "null" {
		enc.AddString("data", r.Data)
	}
	if r.List != "null" {
		enc.AddString("list", r.List)
	}
	if r.Meta != "null" {
		enc.AddString("meta", r.Meta)
	}
	if r.Error != "{}" {
		enc.AddString("error", r.Error)
	}
	return nil
}

type resBody struct {
	Code    int                       `json:"code,omitempty"`
	Status  string                    `json:"status,omitempty"`
	Message string                    `json:"message,omitempty"`
	Data    interface{}               `json:"data,omitempty"`
	List    interface{}               `json:"list,omitempty"`
	Meta    interface{}               `json:"meta,omitempty"`
	Error   common.ErrorResponseModel `json:"error,omitempty"`
}
