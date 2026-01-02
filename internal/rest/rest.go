package rest

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"path"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
)

type ResponseParser[T any] func(response *http.Response) (*T, error)

func GetToken(ctx context.Context, credential *azidentity.DefaultAzureCredential,
	endpoint, path string) (string, error) {

	scope, err := GetURL(endpoint, path)
	if err != nil {
		return "", fmt.Errorf("failed to parse scope: %w", err)
	}

	options := policy.TokenRequestOptions{Scopes: []string{scope.String()}}
	token, err := credential.GetToken(ctx, options)
	if err != nil {
		return "", fmt.Errorf("failed to get token: %w", err)
	}
	return token.Token, nil
}

func GetURL(endpoint string, paths ...string) (*url.URL, error) {
	u, err := url.Parse(fmt.Sprintf("https://%s", endpoint))
	if err != nil {
		return nil, err
	}
	for _, p := range paths {
		u.Path = path.Join(u.Path, p)
	}
	return u, nil
}

func PerformRequest[T any](ctx context.Context, token string, url *url.URL,
	responseParser ResponseParser[T]) (*T, error) {

	response, err := performRequest(ctx, token, url)
	if err != nil {
		return nil, fmt.Errorf("failed to peform request: %v, %w", url, err)
	}
	defer closeBody(response)

	parsed, err := responseParser(response)
	if err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return parsed, nil
}

func performRequest(ctx context.Context, token string, url *url.URL) (*http.Response, error) {

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", token))
	req.Header.Add("x-ms-version", "2020-12-06")

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make http request: %w", err)
	}

	if res.StatusCode < 200 || res.StatusCode >= 300 {
		body, _ := io.ReadAll(res.Body)
		defer closeBody(res)
		slog.Debug("request failed", "status", res.StatusCode, "body", string(body))
		if res.StatusCode == 401 {
			return nil, fmt.Errorf("%w: request failed with status=%d", ErrAuthentication, res.StatusCode)
		}
		return nil, fmt.Errorf("request failed with status=%d", res.StatusCode)
	}
	return res, nil
}

func closeBody(response *http.Response) {
	if response == nil {
		return
	}
	_ = response.Body.Close()
}
