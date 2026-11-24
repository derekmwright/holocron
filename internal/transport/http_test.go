package transport

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
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

func TestHierarchyAndUnknownEnvelopeThroughMCP(t *testing.T) {
	server := testServer(t)
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	client := mcp.NewClient(&mcp.Implementation{Name: "hierarchy-acceptance", Version: "1"}, nil)
	session, err := client.Connect(ctx, &mcp.StreamableClientTransport{Endpoint: server.URL + "/mcp", HTTPClient: &http.Client{Transport: bearer{"curator"}}}, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer session.Close()
	call := func(name string, args any) map[string]any {
		t.Helper()
		result, err := session.CallTool(ctx, &mcp.CallToolParams{Name: name, Arguments: map[string]any{"key": platform.NewID(), "arguments": args}})
		if err != nil || result.IsError {
			t.Fatalf("%s: %#v %v", name, result, err)
		}
		return result.StructuredContent.(map[string]any)
	}
	concept := call("node.create", platform.NodeInput{Category: "concept", Kind: "engineering-discipline", Name: "Spill behavior"})["id"].(string)
	// Omitted legacy scope is essential: missing dimensions must remain unknown,
	// and the MCP schema must permit the explicit v2 envelope instead.
	claim := call("claim.publish", map[string]any{"statement": "Spill mechanism may transfer", "kind": "hypothesis", "limitations": "Unvalidated transfer", "applicability": map[string]any{"version": 2, "state": "unknown", "description": "No broader applicability established"}})["id"].(string)
	association := call("association.propose", platform.AssociationInput{Kind: "claim_concept", Claim: claim, Target: concept, Reason: "Explicit subject"})["id"].(string)
	call("association.review", platform.AssociationReviewInput{ID: association, Verdict: "accepted", Reason: "Curated only"})
	out := call("context.get", map[string]any{"orientation_only": true, "working": map[string]any{"concepts": []string{concept}}, "byte_budget": 16000})
	found := false
	for _, v := range out["cards"].([]any) {
		c := v.(map[string]any)
		if c["id"] == claim {
			found = true
			if c["local_applicability"].(map[string]any)["state"] != "unknown" {
				t.Fatal("unknown became valid")
			}
		}
	}
	if !found {
		t.Fatal("concept discovery failed")
	}
	status, read := request(t, server.URL, "worker-b", "claim.get", platform.IDInput{ID: claim})
	if status != 200 || read["assessment"].(map[string]any)["posture"] != "unassessed" {
		t.Fatal("classification affected epistemics", status, read)
	}
	status, _ = request(t, server.URL, "reviewer", "node.create", platform.NodeInput{Category: "concept", Kind: "topic", Name: "Not an authorized curator"})
	if status != 403 {
		t.Fatal("review role acquired curation", status)
	}
}

func TestWadjetBootstrapIsMetadataOnlyAndRetrySafe(t *testing.T) {
	server := testServer(t)
	namespace := "test-" + platform.NewID()
	script := filepath.Join("..", "..", "scripts", "bootstrap-wadjet.py")
	run := func(token string) (map[string]any, error) {
		t.Helper()
		cmd := exec.Command("python3", script, "--apply", "--url", server.URL, "--namespace", namespace)
		cmd.Env = append(os.Environ(), "HOLOCRON_TOKEN="+token)
		raw, err := cmd.CombinedOutput()
		if err != nil {
			return nil, err
		}
		var out map[string]any
		if err = json.Unmarshal(raw, &out); err != nil {
			t.Fatalf("bootstrap JSON: %v %s", err, raw)
		}
		return out, nil
	}
	if _, err := run("worker-a"); err == nil {
		t.Fatal("bootstrap accepted non-curator")
	}
	first, err := run("curator")
	if err != nil {
		t.Fatal(err)
	}
	second, err := run("curator")
	if err != nil {
		t.Fatal(err)
	}
	a, _ := json.Marshal(first["node_ids"])
	b, _ := json.Marshal(second["node_ids"])
	if !bytes.Equal(a, b) {
		t.Fatal("bootstrap duplicated nodes")
	}
	status, resolved := request(t, server.URL, "worker-b", "context.resolve", map[string]any{"working": first["working"]})
	if status != 200 || len(resolved["direct_concepts"].([]any)) < 2 {
		t.Fatal("bootstrap context unresolved", status, resolved)
	}
	status, bundle := request(t, server.URL, "worker-b", "context.get", first["orientation_request"])
	if status != 200 {
		t.Fatal(status, bundle)
	}
	for _, value := range bundle["cards"].([]any) {
		card := value.(map[string]any)
		if card["local_applicability"].(map[string]any)["state"] != "unknown" {
			t.Fatal("metadata bootstrap established applicability")
		}
	}
}
