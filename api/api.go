package api

type HttpResponse interface {
	GetCode() int32
	GetMessage() string
}
