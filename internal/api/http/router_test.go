package http

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/amakane-hakari/kavos/internal/store"
)

func newTestServer() http.Handler {
	st := store.New[string, string]()
	return NewRouter(st, nil)
}

type successWrap[T any] struct {
	Data T `json:"data"`
}

type kvData struct {
	Key   string `json:"key"`
	Value string `json:"value,omitempty"`
}

type errorWrap struct {
	Error struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

func TestHealth(t *testing.T) {
	ts := httptest.NewServer(newTestServer())
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/health")
	if err != nil {
		t.Fatalf("health request error : %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	var sw successWrap[map[string]string]
	if err := json.NewDecoder(resp.Body).Decode(&sw); err != nil {
		t.Fatalf("decode error: %v", err)
	}
	if sw.Data["status"] != "ok" {
		t.Fatalf("expected status 'ok', got '%v'", sw.Data)
	}
}

func TestKVS_CRUD(t *testing.T) {
	ts := httptest.NewServer(newTestServer())
	defer ts.Close()

	// PUT
	body := bytes.NewBufferString(`{"value":"bar"}`)
	req, _ := http.NewRequest(http.MethodPut, ts.URL+"/kvs/foo", body)
	req.Header.Set("Content-Type", "application/json")
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("put error: %v", err)
	}
	if res.StatusCode != http.StatusOK {
		t.Fatalf("put status %d", res.StatusCode)
	}
	var putResp successWrap[kvData]
	if err := json.NewDecoder(res.Body).Decode(&putResp); err != nil {
		t.Fatalf("decode put: %v", err)
	}
	if putResp.Data.Key != "foo" || putResp.Data.Value != "bar" {
		t.Fatalf("unexpected put data %+v", putResp.Data)
	}

	// GET
	getRes, err := http.Get(ts.URL + "/kvs/foo")
	if err != nil {
		t.Fatalf("get error: %v", err)
	}
	if getRes.StatusCode != http.StatusOK {
		t.Fatalf("get status %d", getRes.StatusCode)
	}
	var getResp successWrap[kvData]
	if err := json.NewDecoder(getRes.Body).Decode(&getResp); err != nil {
		t.Fatalf("get decode error: %v", err)
	}
	if getResp.Data.Value != "bar" {
		t.Fatalf("unexpected value bar got %q", getResp.Data.Value)
	}

	// DELETE
	delReq, _ := http.NewRequest(http.MethodDelete, ts.URL+"/kvs/foo", nil)
	delRes, err := http.DefaultClient.Do(delReq)
	if err != nil {
		t.Fatalf("delete error: %v", err)
	}
	if delRes.StatusCode != http.StatusOK {
		t.Fatalf("delete status %d", delRes.StatusCode)
	}
	var delResp successWrap[kvData]
	if err := json.NewDecoder(delRes.Body).Decode(&delResp); err != nil {
		t.Fatalf("decode delete: %v", err)
	}
	if delResp.Data.Key != "foo" {
		t.Fatalf("unexpected delete key %s", delResp.Data.Key)
	}

	// GET again (not found)
	getRes2, err := http.Get(ts.URL + "/kvs/foo")
	if err != nil {
		t.Fatalf("get2 error: %v", err)
	}
	if getRes2.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", getRes2.StatusCode)
	}
	var errResp errorWrap
	if err := json.NewDecoder(getRes2.Body).Decode(&errResp); err != nil {
		t.Fatalf("decode error: %v", err)
	}
	if errResp.Error.Code != "NOT_FOUND" {
		t.Fatalf("expected NOT_FOUND got %s", errResp.Error.Code)
	}
}

func TestKVS_TTL(t *testing.T) {
	ts := httptest.NewServer(NewRouter(store.New[string, string](), nil))
	defer ts.Close()

	// PUT with ttl=1(1秒)
	reqBody := bytes.NewBufferString(`{"value":"bar"}`)
	req, _ := http.NewRequest(http.MethodPut, ts.URL+"/kvs/tmp?ttl=1", reqBody)
	req.Header.Set("Content-Type", "application/json")
	if resp, err := http.DefaultClient.Do(req); err != nil || resp.StatusCode != http.StatusOK {
		t.Fatalf("put ttl failed: %v code=%v", err, resp.StatusCode)
	}

	// Immediately get (should exist)
	if resp, err := http.Get(ts.URL + "/kvs/tmp"); err != nil || resp.StatusCode != http.StatusOK {
		t.Fatalf("get before expiry failed: %v code=%v", err, resp.StatusCode)
	}

	time.Sleep(1100 * time.Millisecond)

	// After expiry
	resp, err := http.Get(ts.URL + "/kvs/tmp")
	if err != nil {
		t.Fatalf("get after expiry error: %v", err)
	}
	if resp.StatusCode != http.StatusNotFound {
		var dbg map[string]any
		_ = json.NewDecoder(resp.Body).Decode(&dbg)
		t.Fatalf("expected 404 got %d body=%v", resp.StatusCode, dbg)
	}
}