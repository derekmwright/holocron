package transport

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/derekmwright/holocron/internal/platform"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type bearer struct{ token string }

func (b bearer) RoundTrip(r *http.Request) (*http.Response, error) {
	clone := r.Clone(r.Context())
	clone.Header.Set("Authorization", "Bearer "+b.token)
	return http.DefaultTransport.RoundTrip(clone)
}
func testServer(t *testing.T) *httptest.Server {
	t.Helper()
	url := os.Getenv("HOLOCRON_TEST_DATABASE_URL")
	if url == "" {
		t.Skip("run scripts/test-integration.sh")
	}
	s, err := platform.Open(context.Background(), url)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(s.Close)
	server := httptest.NewServer(Handler(s))
	t.Cleanup(server.Close)
	return server
}
func request(t *testing.T, url, token, op string, body any) (int, map[string]any) {
	t.Helper()
	b, _ := json.Marshal(body)
	req, _ := http.NewRequest("POST", url+"/v1/"+op, bytes.NewReader(b))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Idempotency-Key", platform.NewID())
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	var out map[string]any
	if err = json.NewDecoder(resp.Body).Decode(&out); err != nil {
		t.Fatal(err)
	}
	return resp.StatusCode, out
}
func TestRESTAndMCPUseSameAuthorityAndState(t *testing.T) {
	server := testServer(t)
	scope := platform.Scope{Repository: "transport-fixture", Commit: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", Environment: "test", Configuration: "spill", Dataset: "fixture", Oracle: "fixed", Cases: []string{"one"}}
	status, out := request(t, server.URL, "worker-a", "claim.publish", platform.ClaimInput{Statement: "Transport claim", Scope: scope, Kind: "finite_cases", Limitations: "one case"})
	if status != 200 {
		t.Fatal(status, out)
	}
	id := out["id"].(string)
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	connect := func(token string) *mcp.ClientSession {
		t.Helper()
		client := mcp.NewClient(&mcp.Implementation{Name: "test", Version: "1"}, nil)
		session, err := client.Connect(ctx, &mcp.StreamableClientTransport{Endpoint: server.URL + "/mcp", HTTPClient: &http.Client{Transport: bearer{token}}}, nil)
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { session.Close() })
		return session
	}
	session := connect("worker-b")
	tools, err := session.ListTools(ctx, nil)
	if err != nil || len(tools.Tools) != len(platform.Operations) {
		t.Fatalf("tools: %v %v", tools, err)
	}
	result, err := session.CallTool(ctx, &mcp.CallToolParams{Name: "assessment.explain", Arguments: map[string]any{"key": platform.NewID(), "arguments": platform.IDInput{ID: id}}})
	if err != nil || result.IsError {
		t.Fatal(result, err)
	}
	raw, _ := json.Marshal(result.StructuredContent)
	var assessment platform.Assessment
	if err = json.Unmarshal(raw, &assessment); err != nil || assessment.Posture != "unassessed" {
		t.Fatal(string(raw), err)
	}
	denied, err := session.CallTool(ctx, &mcp.CallToolParams{Name: "review.record", Arguments: map[string]any{"key": platform.NewID(), "arguments": platform.ReviewInput{TargetKind: "justification", Target: "missing", Verdict: "accepted", Reason: "self approval"}}})
	if err != nil || !denied.IsError {
		t.Fatal("MCP bypassed authority", denied, err)
	}
	other := connect("outsider")
	hidden, err := other.CallTool(ctx, &mcp.CallToolParams{Name: "claim.get", Arguments: map[string]any{"key": platform.NewID(), "arguments": platform.IDInput{ID: id}}})
	if err != nil || !hidden.IsError {
		t.Fatal("MCP leaked another domain", hidden, err)
	}
	for _, token := range []string{"", "invalid"} {
		status, _ = request(t, server.URL, token, "claim.get", platform.IDInput{ID: id})
		if status != 401 {
			t.Fatal("invalid token accepted", status)
		}
	}
}
func TestHTTPRejectsCrossOriginAndOversizedInput(t *testing.T) {
	server := testServer(t)
	req, _ := http.NewRequest("POST", server.URL+"/v1/work.list", bytes.NewBufferString("{}"))
	req.Header.Set("Authorization", "Bearer worker-a")
	req.Header.Set("Idempotency-Key", platform.NewID())
	req.Header.Set("Origin", "https://untrusted.invalid")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != 403 {
		t.Fatal("cross-origin request accepted", resp.StatusCode)
	}
	status, _ := request(t, server.URL, "worker-a", "claim.publish", map[string]any{"statement": string(bytes.Repeat([]byte{'x'}, 300*1024))})
	if status != 400 {
		t.Fatal("oversized request accepted", status)
	}
}
