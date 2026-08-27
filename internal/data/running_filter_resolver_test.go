package data

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"reflect"
	"strings"
	"testing"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func TestRunningFilterNameResolverUsesFomHotUpdateNames(t *testing.T) {
	var fomEventName string
	ffmCalled := false

	client := &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		var body strings.Builder
		switch req.URL.Path {
		case fomOperatorInfoPath:
			fomEventName = req.URL.Query().Get("event_name")
			if got := req.URL.Query().Get("page_size"); got != "100" {
				t.Fatalf("unexpected FOM page_size: got %q", got)
			}
			_ = json.NewEncoder(&body).Encode(fomOperatorInfoResponse{
				Data: fomOperatorInfoData{
					Refs: []fomOperatorRef{
						{Name: "operator_a"},
						{Name: "operator_b"},
					},
				},
			})
		case ffmFilterNamesPath:
			ffmCalled = true
			t.Fatalf("FFM should not be queried when FOM returns operators")
		default:
			return &http.Response{
				StatusCode: http.StatusNotFound,
				Body:       io.NopCloser(strings.NewReader("not found")),
				Header:     make(http.Header),
			}, nil
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader(body.String())),
			Header:     make(http.Header),
		}, nil
	})}

	resolver := &runningFilterNameResolver{
		fomHost: "http://fom.example",
		ffmURL:  "http://ffm.example",
		client:  client,
	}

	got, err := resolver.ResolveRunningFilterNames(context.Background(), "event_x")
	if err != nil {
		t.Fatalf("ResolveRunningFilterNames returned error: %v", err)
	}

	want := []string{"hotupdate_filter_operator_a", "hotupdate_filter_operator_b"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("filter names = %#v, want %#v", got, want)
	}
	if fomEventName != "event_x" {
		t.Fatalf("FOM event_name = %q, want %q", fomEventName, "event_x")
	}
	if ffmCalled {
		t.Fatal("FFM was queried for hotupdate event")
	}
}

func TestRunningFilterNameResolverUsesFfmWhenFomHasNoOperator(t *testing.T) {
	var ffmEventList []string

	client := &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		var body strings.Builder
		switch req.URL.Path {
		case fomOperatorInfoPath:
			_ = json.NewEncoder(&body).Encode(fomOperatorInfoResponse{
				Data: fomOperatorInfoData{Refs: nil},
			})
		case ffmFilterNamesPath:
			ffmEventList = req.URL.Query()["event_list"]
			_ = json.NewEncoder(&body).Encode(ffmFilterNamesResponse{
				Data: []string{"cpp_filter_a", "cpp_filter_a"},
			})
		default:
			return &http.Response{
				StatusCode: http.StatusNotFound,
				Body:       io.NopCloser(strings.NewReader("not found")),
				Header:     make(http.Header),
			}, nil
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader(body.String())),
			Header:     make(http.Header),
		}, nil
	})}

	resolver := &runningFilterNameResolver{
		fomHost: "http://fom.example",
		ffmURL:  "http://ffm.example",
		client:  client,
	}

	got, err := resolver.ResolveRunningFilterNames(context.Background(), "event_x")
	if err != nil {
		t.Fatalf("ResolveRunningFilterNames returned error: %v", err)
	}

	want := []string{"cpp_filter_a"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("filter names = %#v, want %#v", got, want)
	}
	if !reflect.DeepEqual(ffmEventList, []string{"event_x"}) {
		t.Fatalf("FFM event_list = %#v, want %#v", ffmEventList, []string{"event_x"})
	}
}
