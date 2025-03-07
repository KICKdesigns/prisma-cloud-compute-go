package pcc

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

type Credentials struct {
	ConsoleURL           string `json:"console_url"`
	Username             string `json:"username"`
	Password             string `json:"password"`
	SkipCertVerification bool   `json:"skip_cert_verification"`
}

// A connection to Prisma Cloud Compute.
type Client struct {
	ConsoleURL string
	Username   string
	Password   string
	HTTPClient *http.Client
	JWT        string
}

// Communicate with the Prisma Cloud Compute API.
func (c *Client) Request(method, endpoint string, query, data, response interface{}) (err error) {
	parsedURL, err := url.Parse(c.ConsoleURL)
	if err != nil {
		return err
	}
	parsedEndpoint, err := url.Parse(endpoint)
	if err != nil {
		return err
	}
	completeURL := parsedURL.ResolveReference(parsedEndpoint)

	var buf bytes.Buffer

	if data != nil {
		data_json, err := json.Marshal(data)
		if err != nil {
			return err
		}
		buf = *bytes.NewBuffer(data_json)
	}

	req, err := http.NewRequest(method, completeURL.String(), &buf)
	req.Header.Set("Authorization", "Bearer "+c.JWT)
	req.Header.Set("Content-Type", "application/json")

	res, err := c.HTTPClient.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("Non-OK status: %d", res.StatusCode)
	}

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return err
	}

	if len(body) > 0 {
		if err = json.Unmarshal(body, response); err != nil {
			return err
		}
	}
	return nil
}

// Authenticate with the Prisma Cloud Compute Console.
func (c *Client) authenticate() (err error) {

	type AuthRequest struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}

	type AuthResponse struct {
		Token string `json:"token"`
	}

	res := AuthResponse{}

	if c.Username != "" && c.Password != "" {
		if err := c.Request(http.MethodPost, "/api/v1/authenticate", nil, AuthRequest{c.Username, c.Password}, &res); err != nil {
			return err
		}
	} else {
		return fmt.Errorf("username and/or password missing")
	}

	c.JWT = res.Token
	return nil
}

// Create Client and authenticate.
func APIClient(console_url, username, password string, skip_cert_verification bool) (*Client, error) {
	if !strings.HasSuffix(console_url, "/") {
		console_url += "/"
	}
	apiClient := &Client{
		ConsoleURL: console_url,
		Username:   username,
		Password:   password,
	}

	if skip_cert_verification {
		apiClient.HTTPClient = &http.Client{
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{
					InsecureSkipVerify: true,
				},
			},
		}
	} else {
		apiClient.HTTPClient = &http.Client{}
	}

	if err := apiClient.authenticate(); err != nil {
		return nil, err
	}

	return apiClient, nil
}
