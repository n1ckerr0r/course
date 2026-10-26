package adapter

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
)

func post(ctx context.Context, url, text string) (*http.Response, error) {
	body, _ := json.Marshal(map[string]string{"text": text})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, strings.NewReader(string(body)))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	return http.DefaultClient.Do(req)
}

type HTTPSpamFilter struct{ URL string }

func (f HTTPSpamFilter) IsSpam(ctx context.Context, text string) (bool, error) {
	resp, err := post(ctx, f.URL, text)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()
	return resp.StatusCode == http.StatusForbidden, nil
}

type HTTPToxicityChecker struct{ URL string }

func (c HTTPToxicityChecker) IsToxic(ctx context.Context, text string) (bool, error) {
	resp, err := post(ctx, c.URL, text)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()
	return resp.StatusCode != http.StatusOK, nil
}
