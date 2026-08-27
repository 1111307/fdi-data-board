package data

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"fdi_data_board/internal/biz"
)

const (
	hotUpdateFilterPrefix = "hotupdate_filter_"
	fomOperatorInfoPath   = "/operator/v1/script/info_list"
	ffmFilterNamesPath    = "/fdi_filter/v1/filters/filter_names"
)

type runningFilterNameResolver struct {
	fomHost string
	ffmURL  string
	client  *http.Client
}

func NewRunningFilterNameResolver(data *Data) biz.RunningFilterNameResolver {
	resolver := &runningFilterNameResolver{client: http.DefaultClient}
	if data == nil || data.conf == nil {
		return resolver
	}
	if data.conf.GetFom() != nil {
		resolver.fomHost = strings.TrimRight(data.conf.GetFom().GetHost(), "/")
	}
	if data.conf.GetFfm() != nil {
		resolver.ffmURL = strings.TrimRight(data.conf.GetFfm().GetUrl(), "/")
	}
	return resolver
}

func (r *runningFilterNameResolver) ResolveRunningFilterNames(ctx context.Context, eventName string) ([]string, error) {
	eventName = strings.TrimSpace(eventName)
	if eventName == "" {
		return nil, nil
	}
	if r.fomHost == "" && r.ffmURL == "" {
		return nil, fmt.Errorf("fom.host or ffm.url is required")
	}

	var errs []error
	if r.fomHost != "" {
		names, err := r.queryFomOperatorNames(ctx, eventName)
		if err != nil {
			errs = append(errs, err)
		} else {
			filterNames := make([]string, 0, len(names))
			for _, name := range names {
				name = strings.TrimSpace(name)
				if name == "" {
					continue
				}
				if !strings.HasPrefix(name, hotUpdateFilterPrefix) {
					name = hotUpdateFilterPrefix + name
				}
				filterNames = append(filterNames, name)
			}
			filterNames = uniqueNonEmptyStrings(filterNames)
			if len(filterNames) > 0 {
				return filterNames, nil
			}
		}
	}
	if r.ffmURL != "" {
		names, err := r.queryFfmFilterNames(ctx, eventName)
		if err != nil {
			errs = append(errs, err)
		} else {
			filterNames := uniqueNonEmptyStrings(names)
			if len(filterNames) > 0 {
				return filterNames, nil
			}
		}
	}

	if len(errs) > 0 {
		return nil, fmt.Errorf("resolve running filter names for %q: %w", eventName, errors.Join(errs...))
	}
	return nil, nil
}

type fomOperatorInfoResponse struct {
	Data fomOperatorInfoData `json:"data"`
}

type fomOperatorInfoData struct {
	Refs []fomOperatorRef `json:"refs"`
}

type fomOperatorRef struct {
	Name string `json:"name"`
}

func (r *runningFilterNameResolver) queryFomOperatorNames(ctx context.Context, eventName string) ([]string, error) {
	u, err := url.Parse(r.fomHost + fomOperatorInfoPath)
	if err != nil {
		return nil, err
	}
	q := u.Query()
	q.Set("event_name", eventName)
	q.Set("operator_name", "")
	q.Set("page_size", "100")
	u.RawQuery = q.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := r.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("fom info_list status %d", resp.StatusCode)
	}

	var out fomOperatorInfoResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, err
	}
	names := make([]string, 0, len(out.Data.Refs))
	for _, ref := range out.Data.Refs {
		names = append(names, ref.Name)
	}
	return names, nil
}

type ffmFilterNamesResponse struct {
	Data []string `json:"data"`
}

func (r *runningFilterNameResolver) queryFfmFilterNames(ctx context.Context, eventName string) ([]string, error) {
	u, err := url.Parse(r.ffmURL + ffmFilterNamesPath)
	if err != nil {
		return nil, err
	}
	q := u.Query()
	q.Add("event_list", eventName)
	u.RawQuery = q.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := r.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("ffm filter_names status %d", resp.StatusCode)
	}

	var out ffmFilterNamesResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, err
	}
	return out.Data, nil
}

func uniqueNonEmptyStrings(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	result := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	return result
}
