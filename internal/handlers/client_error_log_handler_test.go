package handlers

import "testing"

func TestExtractTopAppFramePrefersDekorinFrame(t *testing.T) {
	stack := `#0      Object.noSuchMethod (dart:core-patch/object_patch.dart:38:5)
#1      AgendaModel.fromJson (package:dekorin_apps/data/models/agenda_model.dart:42:18)
#2      AgendaService.getAgendas (package:dekorin_apps/data/services/agenda_service.dart:88:21)`

	got := extractTopAppFrame(stack)
	want := "#1      AgendaModel.fromJson (package:dekorin_apps/data/models/agenda_model.dart:42:18)"
	if got != want {
		t.Fatalf("expected %q, got %q", want, got)
	}
}

func TestComputeClientErrorFingerprintStable(t *testing.T) {
	stack := "#1      LoginView.build (package:dekorin_apps/ui/features/login/views/login_view.dart:10:5)"
	fp1 := computeClientErrorFingerprint("android", "TypeError", "/login", stack, "msg 1")
	fp2 := computeClientErrorFingerprint("android", "TypeError", "/login", stack, "msg 2")
	fp3 := computeClientErrorFingerprint("web", "TypeError", "/login", stack, "msg 1")

	if fp1 != fp2 {
		t.Fatalf("expected same fingerprint for same app frame, got %s vs %s", fp1, fp2)
	}
	if fp1 == fp3 {
		t.Fatalf("expected different fingerprint across platforms, got %s", fp1)
	}
}

func TestSanitizeTruncate(t *testing.T) {
	if got := sanitizeTruncate("  hello world  ", 5); got != "hello" {
		t.Fatalf("expected 'hello', got %q", got)
	}
	if got := normalizeLogLevel("FATAL"); got != "fatal" {
		t.Fatalf("expected 'fatal', got %q", got)
	}
	if got := normalizeLogLevel("unknown"); got != "error" {
		t.Fatalf("expected 'error', got %q", got)
	}
}
