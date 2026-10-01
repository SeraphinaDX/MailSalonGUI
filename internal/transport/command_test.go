// SPDX-License-Identifier: GPL-3.0-only

package transport

import (
	"context"
	"strings"
	"testing"
	"time"
)

func TestReceiveAndSend(t *testing.T) {
	out, err := Receive(context.Background(), `printf 'synced'`)
	if err != nil {
		t.Fatal(err)
	}
	if out != "synced" {
		t.Fatalf("receive output = %q", out)
	}

	out, err = Send(context.Background(), `cat`, []byte("message body"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(out) != "message body" {
		t.Fatalf("send stdin was not piped: %q", out)
	}
}

func TestCancelledTransportReturns(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	start := time.Now()
	if _, err := Receive(ctx, "sleep 30"); err == nil {
		t.Fatal("cancelled command succeeded")
	}
	if time.Since(start) > time.Second {
		t.Fatal("cancelled transport blocked")
	}
}
