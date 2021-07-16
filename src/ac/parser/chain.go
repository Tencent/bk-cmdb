package parser

import (
	"errors"
	"fmt"
	"net/http"
	"regexp"

	"configcenter/src/ac/meta"
	"configcenter/src/common"
	"configcenter/src/common/backbone"
	"configcenter/src/common/blog"

	"github.com/tidwall/gjson"
)

type RequestContext struct {
	Rid string
	// http header
	Header http.Header
	// http method
	Method string
	// request's url path
	URI string
	// elements parsed from url, started with api field
	// 0: api field
	// 1: version field
	// 2: action field
	// >=3: resource fields
	Elements []string
	// http request body contents.
	body []byte
	// getBody get http request body contents
	getBody func() (body []byte, err error)
}

// getRequestBody call the callback method to get the request body
func (req *RequestContext) getRequestBody() (body []byte, err error) {
	if req.body == nil {
		body, err = req.getBody()
		if err != nil {
			return
		}
		req.body = body
	}
	return req.body, nil
}

// getValueFromBody get the parameter value from the request body
func (req *RequestContext) getValueFromBody(key string) (value gjson.Result, err error) {
	body, err := req.getRequestBody()
	if err != nil {
		return
	}
	value = gjson.GetBytes(body, key)
	return
}

// getBizIDFromBody get the business id from the request body
func (req *RequestContext) getBizIDFromBody() (biz int64, err error) {
	val, err := req.getValueFromBody(common.BKAppIDField)
	if err != nil {
		return
	}
	biz = val.Int()
	return
}

type parseStream struct {
	RequestCtx *RequestContext
	Attribute  meta.AuthAttribute
	err        error
	action     meta.Action
	engine     *backbone.Engine
}

func newParseStream(rc *RequestContext, engine *backbone.Engine) (*parseStream, error) {
	if nil == rc {
		return nil, errors.New("request context is nil")
	}

	return &parseStream{RequestCtx: rc, engine: engine}, nil
}

// parse is used to parse the auth attribute from RequestContext.
func (ps *parseStream) Parse() (*meta.AuthAttribute, error) {
	if ps.err != nil {
		return nil, ps.err
	}

	ps.validateAPI().
		validateVersion().
		validateResourceAction().
		validateUserAndSupplier().
		matchAuthRoute().
		finalizer()

	if ps.err != nil {
		return nil, ps.err
	}

	for index := range ps.Attribute.Resources {
		ps.Attribute.Resources[index].SupplierAccount = ps.Attribute.User.SupplierAccount
	}

	return &ps.Attribute, nil
}

func (ps *parseStream) validateAPI() *parseStream {
	if ps.shouldReturn() {
		return ps
	}

	if ps.RequestCtx.Elements[0] != "api" {
		ps.err = errors.New("unsupported api format")
	}

	return ps
}

func (ps *parseStream) validateVersion() *parseStream {
	if ps.shouldReturn() {
		return ps
	}

	version := ps.RequestCtx.Elements[1]
	if version != "v3" {
		ps.err = fmt.Errorf("unsupported version %s", version)
		return ps
	}

	return ps
}

func (ps *parseStream) validateResourceAction() *parseStream {
	if ps.shouldReturn() {
		return ps
	}

	action := ps.RequestCtx.Elements[2]
	switch action {
	case "find":
		ps.action = meta.Find
	case "findMany":
		ps.action = meta.FindMany

	case "create":
		ps.action = meta.Create
	case "createMany":
		ps.action = meta.CreateMany

	case "update":
		ps.action = meta.Update
	case "updateMany":
		ps.action = meta.UpdateMany

	case "delete":
		ps.action = meta.Delete
	case "deleteMany":
		ps.action = meta.DeleteMany

	default:
		ps.action = meta.Unknown
		// to compatible api that is not this kind of format,
		// this err will not be set, but it will be set when
		// all the api is normalized.

		// TODO: uncomment this err code.
		// ps.err = fmt.Errorf("unsupported action %s", action)
		return ps
	}

	return ps
}

// user and supplier account must be set in the http
// request header, otherwise, an error will be occur.
func (ps *parseStream) validateUserAndSupplier() *parseStream {
	if ps.shouldReturn() {
		return ps
	}

	// validate user header at first.
	user := ps.RequestCtx.Header.Get(common.BKHTTPHeaderUser)
	if len(user) == 0 {
		ps.err = fmt.Errorf("request lost header: %s", common.BKHTTPHeaderUser)
		return ps
	}
	ps.Attribute.User.UserName = user

	// validate the supplier account now.
	supplier := ps.RequestCtx.Header.Get(common.BKHTTPOwnerID)
	if len(supplier) == 0 {
		ps.err = fmt.Errorf("request lost header: %s", common.BKHTTPOwnerID)
		return ps
	}
	ps.Attribute.User.SupplierAccount = supplier

	return ps
}

// finalizer is to find whether a url resource has been matched or not.
func (ps *parseStream) finalizer() *parseStream {
	if ps.shouldReturn() {
		return ps
	}
	if len(ps.Attribute.Resources) <= 0 {
		ps.err = errors.New("no matched rules")
	}
	return ps
}

func (ps *parseStream) shouldReturn() bool {
	return ps.err != nil || len(ps.Attribute.Resources) > 0
}

func (ps *parseStream) hitRegexp(reg *regexp.Regexp, httpMethod string) bool {
	result := reg.MatchString(ps.RequestCtx.URI) && ps.RequestCtx.Method == httpMethod
	if result {
		blog.V(4).Infof("match %s %s", httpMethod, reg)
	}
	return result
}

func (ps *parseStream) hitPattern(pattern, httpMethod string) bool {
	result := pattern == ps.RequestCtx.URI && ps.RequestCtx.Method == httpMethod
	if result {
		blog.V(4).Infof("match %s %s", httpMethod, pattern)
	}
	return result
}
