package main

import (
	"encoding/json"
	log "github.com/cihub/seelog"
	"net/http"
	"net/url"
	"time"
)

type Response struct {
	w      http.ResponseWriter
	size   int
	status int
	start  time.Time
}

func NewResponse(w http.ResponseWriter) *Response {
	return &Response{w: w, size: 0, status: http.StatusOK, start: time.Now()}
}

func (r *Response) Header() http.Header {
	return r.w.Header()
}

func (r *Response) Write(b []byte) (int, error) {
	result, err := r.w.Write(b)
	r.size += result
	return result, err
}

func (r *Response) WriteHeader(i int) {
	r.status = i
	r.w.WriteHeader(i)
}

type LogItem struct {
	Protocol     string        `json:"protocol,omitempty"`
	Method       string        `json:"method,omitempty"`
	Status       int           `json:"status,omitempty"`
	URI          string        `json:"uri,omitempty"`
	PostData     url.Values    `json:"post_data,omitempty"`
	Host         string        `json:"host,omitempty"`
	UserAgent    string        `json:"uset-agent,omitempty"`
	RemoteAddr   string        `json:"remote-addr,omitempty"`
	ResponseSize int           `json:"response-size,omitempty"`
	Elapse       time.Duration `json:"elapse,omitempty"`
}

func (i *LogItem) String() string {
	byte, _ := json.Marshal(i)
	return string(byte)
}

func loggingAccessLog(req *http.Request, res *Response) {
	i := &LogItem{
		Protocol:     req.Proto,
		Method:       req.Method,
		Status:       res.status,
		URI:          req.RequestURI,
		PostData:     req.PostForm,
		Host:         req.Host,
		UserAgent:    req.UserAgent(),
		RemoteAddr:   req.RemoteAddr,
		ResponseSize: res.size,
		Elapse:       time.Now().Sub(res.start),
	}

	log.Info("Webhook accessLog: ", i)
}
