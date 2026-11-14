package mailer

import "testing"

func TestEnvelopeAddr(t *testing.T) {
	m := New("h", "25", "", "", "Физтех.Ответ <no-reply@phystech.answer>")
	if got := m.envelopeAddr(); got != "no-reply@phystech.answer" {
		t.Fatalf("envelopeAddr = %q", got)
	}
	m = New("h", "25", "", "", "plain@example.com")
	if got := m.envelopeAddr(); got != "plain@example.com" {
		t.Fatalf("bare envelopeAddr = %q", got)
	}
}

func TestFromHeader(t *testing.T) {
	m := New("h", "25", "", "", "Физтех.Ответ <no-reply@phystech.answer>")
	got := m.fromHeader()
	// Кириллическое имя → RFC 2047 encoded-word, адрес остаётся читаемым.
	if got == "Физтех.Ответ <no-reply@phystech.answer>" {
		t.Fatal("cyrillic display name must be encoded")
	}
	if want := "=?UTF-8?B?"; len(got) < len(want) || got[:len(want)] != want {
		t.Fatalf("fromHeader = %q", got)
	}
	if last := got[len(got)-len("no-reply@phystech.answer>"):]; last != "no-reply@phystech.answer>" {
		t.Fatalf("address part lost: %q", got)
	}
}

func TestFromHeaderASCII(t *testing.T) {
	m := New("h", "25", "", "", "Phystech <no-reply@phystech.answer>")
	if got, want := m.fromHeader(), "Phystech <no-reply@phystech.answer>"; got != want {
		t.Fatalf("ascii fromHeader = %q, want %q", got, want)
	}
}
