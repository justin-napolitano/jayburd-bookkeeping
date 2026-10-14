package observer

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"
)

var projectID = regexp.MustCompile(`^[a-z][a-z0-9-]{4,28}[a-z0-9]$`)
var locationID = regexp.MustCompile(`^[a-z]+-[a-z]+[0-9]+$`)
var queueID = regexp.MustCompile(`^[A-Za-z0-9-]{1,100}$`)

// AccessTokenProvider supplies short-lived OAuth credentials and must honor context.
type AccessTokenProvider interface {
	AccessToken(context.Context) (string, error)
}

// QueueConfig fixes one destination and invocation identity, never request-controlled routing.
type QueueConfig struct{ Project, Location, Queue, ConsumerURL, Audience, ServiceAccount string }

// CloudTasksTransport makes one bounded REST create call, with no internal retries.
type CloudTasksTransport struct {
	config QueueConfig
	parent string
	tokens AccessTokenProvider
	client *http.Client
}

// NewCloudTasksTransport validates routing and injects credentials without retrieving them.
func NewCloudTasksTransport(c QueueConfig, tokens AccessTokenProvider, transport http.RoundTripper) (*CloudTasksTransport, error) {
	target, ok := runURL(c.ConsumerURL)
	audience, aok := runURL(c.Audience)
	if tokens == nil || transport == nil || !ok || !aok || target.Path != ConsumerPath || (audience.Path != "" && audience.Path != "/") || target.Host != audience.Host || !projectID.MatchString(c.Project) || !locationID.MatchString(c.Location) || !queueID.MatchString(c.Queue) || !serviceEmail.MatchString(c.ServiceAccount) || !strings.HasSuffix(c.ServiceAccount, "@"+c.Project+".iam.gserviceaccount.com") {
		return nil, reject()
	}
	return &CloudTasksTransport{c, "projects/" + c.Project + "/locations/" + c.Location + "/queues/" + c.Queue, tokens, &http.Client{Transport: transport, Timeout: 10 * time.Second, CheckRedirect: noRedirect}}, nil
}

// Enqueue validates the stable identity before credentials and keeps uncertainty distinct from completion.
func (t *CloudTasksTransport) Enqueue(parent context.Context, task Task) (EnqueueResult, error) {
	w, e := ParseWakeup(task.Body)
	if e != nil || task.TargetVersion != 1 || task.Name != taskName(w.OperationID, w.DispatchGeneration) {
		return "", reject()
	}
	ctx, cancel := context.WithTimeout(parent, 10*time.Second)
	defer cancel()
	if ctx.Err() != nil {
		return Timeout, nil
	}
	token, e := t.tokens.AccessToken(ctx)
	if ctx.Err() != nil {
		return Timeout, nil
	}
	if e != nil {
		return Transient, nil
	}
	if token == "" || len(token) > 8192 || strings.ContainsAny(token, " \t\r\n") {
		return Rejected, nil
	}
	name := t.parent + "/tasks/" + task.Name
	raw, _ := json.Marshal(map[string]any{"responseView": "BASIC", "task": map[string]any{
		"name": name, "dispatchDeadline": "65s", "httpRequest": map[string]any{
			"url": t.config.ConsumerURL, "httpMethod": "POST", "headers": map[string]string{"Content-Type": "application/json"}, "body": task.Body,
			"oidcToken": map[string]string{"serviceAccountEmail": t.config.ServiceAccount, "audience": t.config.Audience},
		},
	}})
	req, e := http.NewRequestWithContext(ctx, http.MethodPost, "https://cloudtasks.googleapis.com/v2/"+t.parent+"/tasks", bytes.NewReader(raw))
	if e != nil {
		return "", reject()
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	resp, e := t.client.Do(req)
	if e != nil {
		if ctx.Err() != nil {
			return Timeout, nil
		}
		return Transient, nil
	}
	defer resp.Body.Close()
	if ctx.Err() != nil {
		return Timeout, nil
	}
	switch {
	case resp.StatusCode == 409:
		return AlreadyExists, nil
	case resp.StatusCode == 408 || resp.StatusCode == 504:
		return Timeout, nil
	case resp.StatusCode == 429 || resp.StatusCode >= 500:
		return Transient, nil
	case resp.StatusCode >= 400:
		return Rejected, nil
	case resp.StatusCode != 200 && resp.StatusCode != 201:
		return Transient, nil
	}
	body, e := io.ReadAll(io.LimitReader(resp.Body, 16385))
	if ctx.Err() != nil {
		return Timeout, nil
	}
	if e != nil || len(body) > 16384 {
		return Transient, nil
	}
	var result struct {
		Name string `json:"name"`
	}
	if json.Unmarshal(body, &result) != nil || result.Name != name {
		return Transient, nil
	}
	return Accepted, nil
}
