package grafsdk

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

type Client struct {
	apiURL     string
	apiKey     string
	httpClient *HTTPClient
}

type SearchOption func(values *url.Values)

func FolderTypeSearchOption() SearchOption {
	return func(values *url.Values) {
		values.Set("type", string(DashHitFolder))
	}
}

func DashTypeSearchOption() SearchOption {
	return func(values *url.Values) {
		values.Set("type", string(DashHitDB))
	}
}

func QuerySearchOption(q string) SearchOption {
	return func(values *url.Values) {
		values.Set("query", q)
	}
}

func FolderIDsSearchOption(ids []int64) SearchOption {
	return func(values *url.Values) {
		idsStr := make([]string, 0, len(ids))
		for _, id := range ids {
			idsStr = append(idsStr, strconv.FormatInt(id, 10))
		}
		values.Set("folderIds", strings.Join(idsStr, ","))
	}
}

func New(apiURL string, apiKey string) *Client {
	httpc := NewHTTPClient(context.Background())
	httpc.SetHeaders(map[string]string{
		"Authorization": fmt.Sprintf("Bearer %s", apiKey),
		"Accept":        "application/json",
		"Content-Type":  "application/json",
	})
	return &Client{
		apiURL:     apiURL,
		apiKey:     apiKey,
		httpClient: httpc,
	}
}

func (c *Client) SaveDashboard(ctx context.Context, payload *DashboardSavePayload) error {
	by, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, fmt.Sprintf("%s/api/dashboards/db", c.apiURL), bytes.NewReader(by))
	if err != nil {
		return fmt.Errorf("NewRequestWithContext: %w", err)
	}
	resp, _, err := c.do(ctx, req, nil)
	if err != nil {
		return fmt.Errorf("do: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("status code: %d", resp.StatusCode)
	}

	return nil
}

func (c *Client) GetDashboardByUID(ctx context.Context, uid string) (*DashboardWithMeta, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fmt.Sprintf("%s/api/dashboards/uid/%s", c.apiURL, uid), nil)
	if err != nil {
		return nil, fmt.Errorf("NewRequestWithContext: %w", err)
	}
	dashboard := &DashboardWithMeta{}
	resp, _, err := c.do(ctx, req, dashboard)
	if err != nil {
		return nil, fmt.Errorf("do: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("status code: %d", resp.StatusCode)
	}

	return dashboard, nil
}

func (c *Client) CreateFolder(ctx context.Context, title string) (*Folder, error) {
	folder := Folder{Title: title}
	folderBy, err := json.Marshal(folder)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, fmt.Sprintf("%s/api/folders", c.apiURL), bytes.NewReader(folderBy))
	if err != nil {
		return nil, fmt.Errorf("NewRequestWithContext: %w", err)
	}
	resp, _, err := c.do(ctx, req, &folder)
	if err != nil {
		return nil, fmt.Errorf("do: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("status code: %d", resp.StatusCode)
	}

	return &folder, nil
}

func (c *Client) ListFolders(ctx context.Context) ([]*Folder, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fmt.Sprintf("%s/api/folders", c.apiURL), nil)
	if err != nil {
		return nil, fmt.Errorf("NewRequestWithContext: %w", err)
	}
	folders := []*Folder{}
	resp, _, err := c.do(ctx, req, &folders)
	if err != nil {
		return nil, fmt.Errorf("do: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("status code: %d", resp.StatusCode)
	}

	return folders, nil
}

func (c *Client) CreateDatasource(ctx context.Context, datasource *Datasource) (*Datasource, error) {
	if datasource == nil {
		return nil, fmt.Errorf("missing datasource")
	}
	by, err := json.Marshal(datasource)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, fmt.Sprintf("%s/api/datasources", c.apiURL), bytes.NewReader(by))
	if err != nil {
		return nil, fmt.Errorf("NewRequestWithContext: %w", err)
	}

	datasourceResp := Datasource{}
	resp, _, err := c.do(ctx, req, &datasourceResp)
	if err != nil {
		return nil, fmt.Errorf("do: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("status code: %d", resp.StatusCode)
	}

	return &datasourceResp, nil
}

func (c *Client) UpdateDatasource(ctx context.Context, datasource *Datasource) error {
	if datasource == nil {
		return fmt.Errorf("missing datasource")
	}
	by, err := json.Marshal(datasource)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPut, fmt.Sprintf("%s/api/datasources/%d", c.apiURL, datasource.ID), bytes.NewReader(by))
	if err != nil {
		return fmt.Errorf("NewRequestWithContext: %w", err)
	}
	resp, _, err := c.do(ctx, req, nil)
	if err != nil {
		return fmt.Errorf("do: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("status code: %d", resp.StatusCode)
	}

	return nil
}

func (c *Client) GetDatasourceByID(ctx context.Context, id int64) (*Datasource, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fmt.Sprintf("%s/api/datasources/%d", c.apiURL, id), nil)
	if err != nil {
		return nil, fmt.Errorf("NewRequestWithContext: %w", err)
	}
	datasource := Datasource{}
	resp, _, err := c.do(ctx, req, &datasource)
	if err != nil {
		return nil, fmt.Errorf("do: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("status code: %d", resp.StatusCode)
	}

	return &datasource, nil
}

func (c *Client) GetDatasourceByName(ctx context.Context, name string) (*Datasource, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fmt.Sprintf("%s/api/datasources/name/%s", c.apiURL, name), nil)
	if err != nil {
		return nil, fmt.Errorf("NewRequestWithContext: %w", err)
	}
	datasource := Datasource{}
	resp, _, err := c.do(ctx, req, &datasource)
	if err != nil {
		return nil, fmt.Errorf("do: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("status code: %d", resp.StatusCode)
	}

	return &datasource, nil
}

func (c *Client) ListDatasources(ctx context.Context) ([]*Datasource, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fmt.Sprintf("%s/api/datasources", c.apiURL), nil)
	if err != nil {
		return nil, fmt.Errorf("NewRequestWithContext: %w", err)
	}
	datasources := []*Datasource{}
	resp, _, err := c.do(ctx, req, &datasources)
	if err != nil {
		return nil, fmt.Errorf("do: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("status code: %d", resp.StatusCode)
	}

	return datasources, nil
}

func (c *Client) Search(ctx context.Context, searchOptions ...SearchOption) ([]*SearchResult, error) {
	u, err := url.Parse(fmt.Sprintf("%s/api/search", c.apiURL))
	if err != nil {
		return nil, fmt.Errorf("url.Parse: %w", err)
	}
	urlValues := url.Values{}
	for _, searchOption := range searchOptions {
		searchOption(&urlValues)
	}
	u.RawQuery = urlValues.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("NewRequestWithContext: %w", err)
	}
	searchResults := []*SearchResult{}
	resp, _, err := c.do(ctx, req, &searchResults)
	if err != nil {
		return nil, fmt.Errorf("do: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("status code: %d", resp.StatusCode)
	}

	return searchResults, nil
}

func (a *Client) do(ctx context.Context, req *http.Request, respData interface{}) (*http.Response, []byte, error) {
	resp, err := a.httpClient.Do(req)
	if err != nil {
		return nil, nil, fmt.Errorf("Do: %w", err)
	}

	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return resp, nil, fmt.Errorf("io.ReadAll: %w", err)
	}

	if respData != nil {
		if err := json.Unmarshal(body, respData); err != nil {
			return resp, nil, fmt.Errorf("json.Unmarshal: %w", err)
		}
	}

	return resp, body, nil
}
