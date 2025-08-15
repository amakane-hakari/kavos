package http

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/amakane-hakari/kavos/internal/store"
)

func newTestServer() http.Handler {
	st := store.New()
	return NewRouter(st)
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

	// GET
	getRes, err := http.Get(ts.URL + "/kvs/foo")
	if err != nil {
		t.Fatalf("get error: %v", err)
	}
	if getRes.StatusCode != http.StatusOK {
		t.Fatalf("get status %d", getRes.StatusCode)
	}
	var vr valueResponse
	if err := json.NewDecoder(getRes.Body).Decode(&vr); err != nil {
		t.Fatalf("get decode error: %v", err)
	}
	if vr.Value != "bar" {
		t.Fatalf("expected value 'bar', got '%s'", vr.Value)
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

	// GET again (not found)
	getRes2, err := http.Get(ts.URL + "/kvs/foo")
	if err != nil {
		t.Fatalf("get2 error: %v", err)
	}
	if getRes2.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", getRes2.StatusCode)
	}
}
