package tools

import (
	"bytes"
	"crypto/md5"
	"encoding/hex"
	"github.com/coolwhite520/alipayserver/ip"
	"github.com/kataras/iris"
	log "github.com/sirupsen/logrus"
	"io/ioutil"
	"net/http"
	"reflect"
	"sort"
	"strconv"
	"time"
)

const AppSecret = "75a204a606a1eac12d068c8c0473ba662575d0bf278f845a741728b9dca157a4"

// GetSign get the sign info
func GetSign(data interface{}, appSecret string) string {
	md5ctx := md5.New()
	switch v := reflect.ValueOf(data); v.Kind() {
	case reflect.String:
		md5ctx.Write([]byte(v.String() + appSecret))
		return hex.EncodeToString(md5ctx.Sum(nil))
	case reflect.Struct:
		orderStr := StructToMapSing(v.Interface(), appSecret)
		md5ctx.Write([]byte(orderStr))
		return hex.EncodeToString(md5ctx.Sum(nil))
	case reflect.Ptr:
		originType := v.Elem().Type()
		if originType.Kind() != reflect.Struct {
			return ""
		}
		dataType := reflect.TypeOf(data).Elem()
		dataVal := v.Elem()
		orderStr := buildOrderStr(dataType, dataVal, appSecret)
		md5ctx.Write([]byte(orderStr))
		return hex.EncodeToString(md5ctx.Sum(nil))
	default:
		return ""
	}
}
func buildOrderStr(t reflect.Type, v reflect.Value, appSecret string) (returnStr string) {
	keys := make([]string, 0, t.NumField())

	var data = make(map[string]interface{})
	for i := 0; i < t.NumField(); i++ {
		kName := t.Field(i).Tag.Get("json")
		if  kName == "sign" || kName == "pareacode" || kName == "msg" || kName == "careacode"{
			continue
		}
		data[t.Field(i).Tag.Get("json")] = v.Field(i).Interface()

		keys = append(keys, t.Field(i).Tag.Get("json"))
	}

	sort.Sort(sort.StringSlice(keys))

	var buf bytes.Buffer
	for _, k := range keys {
		if data[k] == "" {
			continue
		}
		if buf.Len() > 0 {
			buf.WriteByte('&')
		}

		buf.WriteString(k)
		buf.WriteByte('=')
		switch vv := data[k].(type) {
		case string:
			buf.WriteString(vv)
		case int:
		case int8:
		case int16:
		case int32:
		case int64:
			buf.WriteString(strconv.FormatInt(int64(vv), 10))
		default:
			continue
		}
	}

	buf.WriteString("&key=" + appSecret)
	returnStr = buf.String()

	return returnStr
}

func StructToMapSing(content interface{}, appSecret string) (returnStr string) {

	t := reflect.TypeOf(content)
	v := reflect.ValueOf(content)

	returnStr = buildOrderStr(t, v, appSecret)

	return returnStr
}

// LoggerMiddleware 日志中间件
func LoggerMiddleware(ctx iris.Context) {
	p := ctx.Request().URL.Path
	method := ctx.Request().Method
	start := time.Now()
	fields := make(map[string]interface{})
	fields["title"] = "G@@dL#ck."
	fields["ip"] = ip.RemoteIp(ctx.Request())
	fields["method"] = method
	fields["url"] = ctx.Request().URL.String()
	//fields["proto"] = ctx.Request().Proto
	//fields["header"] = ctx.Request().Header
	fields["user_agent"] = ctx.Request().UserAgent()
	//fields["x_request_id"] = ctx.GetHeader("X-Request-Id")

	// 如果是POST/PUT请求，并且内容类型为JSON，则读取内容体
	if method == http.MethodPost || method == http.MethodPut || method == http.MethodPatch {
		body, err := ioutil.ReadAll(ctx.Request().Body)
		if err == nil {
			defer ctx.Request().Body.Close()
			buf := bytes.NewBuffer(body)
			ctx.Request().Body = ioutil.NopCloser(buf)
			//fields["content_length"] = ctx.GetContentLength()
			//fields["body"] = string(body)
		}
	}
	log.WithFields(fields).Infof("[http] %s-%s-%s-%d(0ms)",
		p, ctx.Request().Method, ip.RemoteIp(ctx.Request()), ctx.ResponseWriter().StatusCode())
	ctx.Next()

	//下面是返回日志
	fields["res_status"] = ctx.ResponseWriter().StatusCode()
	if ctx.Values().GetString("out_err") != "" {
		fields["out_err"] = ctx.Values().GetString("out_err")
	}
	fields["res_length"] = ctx.ResponseWriter().Header().Get("size")
	if v := ctx.Values().Get("res_body"); v != nil {
		if b, ok := v.([]byte); ok {
			fields["res_body"] = string(b)
		}
	}
	fields["uid"] = ctx.Values().GetString("uid")
	timeConsuming := time.Since(start).Nanoseconds() / 1e6
	log.WithFields(fields).Infof("[http] %s-%s-%s-%d(%dms)",
		p, ctx.Request().Method, ip.RemoteIp(ctx.Request()), ctx.ResponseWriter().StatusCode(), timeConsuming)
}
