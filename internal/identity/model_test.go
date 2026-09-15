package identity

import "testing"

func TestPolicyRejectsUnknownAndRequiresSelection(t *testing.T) {
	for _, preset := range []string{"", "read-only", "collaborate", "maintain"} {
		p := Policy{Preset: preset, Exceptions: map[string]bool{"token": true}}
		if p.Allows("token") {
			t.Fatal("unknown permission authorized")
		}
	}
	if (Policy{}).Allows("read") {
		t.Fatal("missing policy grants access")
	}
	p := Policy{Preset: "maintain", Exceptions: map[string]bool{"merge": false}}
	if p.Allows("merge") || !p.Allows("read") || p.Allows("persona") {
		t.Fatal("incorrect exceptions or preset")
	}
}

func TestPersonaValidation(t *testing.T) {
	p := Persona{ID: "work", Name: "Work", GitName: "Example", GitEmail: "example@example.test", Host: "github.com", Account: "example"}
	if err := p.Validate(); err != nil {
		t.Fatal(err)
	}
	for _, host := range []string{"github.com/path", "localhost:80", "--hostname", "evil..com"} {
		p.Host = host
		if p.Validate() == nil {
			t.Fatalf("accepted %s", host)
		}
	}
}
