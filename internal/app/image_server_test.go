package app

import "testing"

func TestImageServerService_SetImgServerHost(t *testing.T) {
	svc := NewImageServerService("", "")
	if got := svc.GetImgServerHost(); got != "localhost:"+defaultImagePort {
		t.Fatalf("got=%q", got)
	}

	svc.SetImgServerHost(" ")
	if got := svc.GetImgServerHost(); got != "localhost:"+defaultImagePort {
		t.Fatalf("got=%q", got)
	}

	svc.SetImgServerHost("example.com")
	if got := svc.GetImgServerHost(); got != "example.com:"+defaultImagePort {
		t.Fatalf("got=%q", got)
	}
}
