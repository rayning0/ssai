package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestProgramAdFragmentLoop(t *testing.T) {
	if currentSequence(0) != 0 || currentSequence(29*time.Second) != 4 || currentSequence(30*time.Second) != 5 || currentSequence(36*time.Second) != 6 || currentSequence(40*time.Second) != 7 {
		t.Fatal("unexpected fragment schedule")
	}
	if assetForSequence(4).isAd || !assetForSequence(5).isAd || !assetForSequence(6).isAd {
		t.Fatal("incorrect ad slots")
	}
	if assetForSequence(6).duration != 4*time.Second {
		t.Fatal("last ad fragment should be four seconds")
	}
}

func TestCORSIncludesRangeRequests(t *testing.T) {
	recorder := httptest.NewRecorder()
	cors(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusPartialContent) })).ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/segments/1.ts", nil))
	if recorder.Header().Get("Access-Control-Allow-Origin") != "http://localhost:8080" {
		t.Fatal("missing CORS origin header")
	}
	if recorder.Header().Get("Access-Control-Allow-Headers") != "Range" {
		t.Fatal("missing Range CORS header")
	}
}

func TestSequenceFromPath(t *testing.T) {
	if sequence, ok := sequenceFromPath("/segments/17.ts"); !ok || sequence != 17 {
		t.Fatal("valid segment path was rejected")
	}
	if _, ok := sequenceFromPath("/segments/nope.ts"); ok {
		t.Fatal("invalid segment path was accepted")
	}
}
