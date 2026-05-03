package internal

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
)

type SupabaseClient struct {
	URL  string
	Key  string
	HTTP *http.Client
}

func InitSupabase() *SupabaseClient {
	return &SupabaseClient{
		URL:  os.Getenv("SUPABASE_URL"),
		Key:  os.Getenv("SUPABASE_KEY"),
		HTTP: &http.Client{},
	}
}

func (sc *SupabaseClient) addAuthHeaders(req *http.Request) {
	req.Header.Set("Authorization", "Bearer "+sc.Key)
	req.Header.Set("apikey", sc.Key)
	req.Header.Set("Content-Type", "application/json")
}

func decodeSupabaseResponse(resp *http.Response, target interface{}) error {
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		if len(body) == 0 {
			return fmt.Errorf("supabase request failed: %s", resp.Status)
		}
		return fmt.Errorf("supabase request failed: %s: %s", resp.Status, string(body))
	}

	if target == nil {
		return nil
	}

	return json.NewDecoder(resp.Body).Decode(target)
}

// SelectFrom - Generic SELECT query with filters
func (sc *SupabaseClient) SelectFrom(table string, selectCols string, filters map[string]interface{}) ([]map[string]interface{}, error) {
	u, _ := url.Parse(fmt.Sprintf("%s/rest/v1/%s", sc.URL, table))
	q := u.Query()
	q.Set("select", selectCols)

	for key, val := range filters {
		q.Set(key, fmt.Sprintf("eq.%v", val))
	}

	u.RawQuery = q.Encode()
	req, _ := http.NewRequest("GET", u.String(), nil)
	sc.addAuthHeaders(req)

	resp, err := sc.HTTP.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var data []map[string]interface{}
	if err := decodeSupabaseResponse(resp, &data); err != nil {
		return nil, err
	}
	return data, nil
}

// InsertInto - Generic INSERT query
func (sc *SupabaseClient) InsertInto(table string, data interface{}) ([]map[string]interface{}, error) {
	url := fmt.Sprintf("%s/rest/v1/%s", sc.URL, table)
	jsonBody, _ := json.Marshal(data)

	req, _ := http.NewRequest("POST", url, bytes.NewBuffer(jsonBody))
	sc.addAuthHeaders(req)
	req.Header.Set("Prefer", "return=representation")

	resp, err := sc.HTTP.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result []map[string]interface{}
	if err := decodeSupabaseResponse(resp, &result); err != nil {
		return nil, err
	}
	return result, nil
}

// UpdateTable - Generic UPDATE query
func (sc *SupabaseClient) UpdateTable(table string, data map[string]interface{}, filters map[string]interface{}) error {
	u, _ := url.Parse(fmt.Sprintf("%s/rest/v1/%s", sc.URL, table))
	q := u.Query()

	for key, val := range filters {
		q.Set(key, fmt.Sprintf("eq.%v", val))
	}

	u.RawQuery = q.Encode()
	jsonBody, _ := json.Marshal(data)

	req, _ := http.NewRequest("PATCH", u.String(), bytes.NewBuffer(jsonBody))
	sc.addAuthHeaders(req)

	resp, err := sc.HTTP.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return decodeSupabaseResponse(resp, nil)
}

// DeleteFrom - Generic DELETE query
func (sc *SupabaseClient) DeleteFrom(table string, filters map[string]interface{}) error {
	u, _ := url.Parse(fmt.Sprintf("%s/rest/v1/%s", sc.URL, table))
	q := u.Query()

	for key, val := range filters {
		q.Set(key, fmt.Sprintf("eq.%v", val))
	}

	u.RawQuery = q.Encode()

	req, _ := http.NewRequest("DELETE", u.String(), nil)
	sc.addAuthHeaders(req)

	resp, err := sc.HTTP.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return decodeSupabaseResponse(resp, nil)
}

// Query example: Get all users
func (sc *SupabaseClient) GetUsers() ([]map[string]interface{}, error) {
	url := fmt.Sprintf("%s/rest/v1/users?select=*", sc.URL)
	req, _ := http.NewRequest("GET", url, nil)
	sc.addAuthHeaders(req)

	resp, err := sc.HTTP.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var data []map[string]interface{}
	if err := decodeSupabaseResponse(resp, &data); err != nil {
		return nil, err
	}
	return data, nil
}

// Insert example
func (sc *SupabaseClient) InsertUser(email, name string) error {
	url := fmt.Sprintf("%s/rest/v1/users", sc.URL)
	body := map[string]string{"email": email, "name": name}
	jsonBody, _ := json.Marshal(body)

	req, _ := http.NewRequest("POST", url, bytes.NewBuffer(jsonBody))
	sc.addAuthHeaders(req)

	resp, err := sc.HTTP.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return decodeSupabaseResponse(resp, nil)
}
