package identityui

import (
	"context"
	"errors"
	"io"

	"charm.land/huh/v2"
	"github.com/MrMaxie/dovik/internal/identity"
	"github.com/MrMaxie/dovik/internal/operatorclient"
	"github.com/MrMaxie/dovik/internal/terminalstyle"
)

// EditPersona updates public persona metadata without exposing credentials.
func EditPersona(ctx context.Context, client operatorclient.Client, personaID string, input io.Reader, output io.Writer) error {
	err := editPersona(ctx, client, personaID, input, output)
	if errors.Is(err, huh.ErrUserAborted) {
		return nil
	}
	return err
}

func editPersona(ctx context.Context, client operatorclient.Client, personaID string, input io.Reader, output io.Writer) error {
	snapshot, err := client.Identity(ctx, identity.Request{Action: "get"})
	if err != nil {
		return err
	}
	persona, err := snapshot.State.FindPersona(personaID)
	if err != nil {
		return err
	}
	form := func(fields ...huh.Field) error {
		return terminalstyle.NewForm("Edit persona", input, output, fields...).RunWithContext(ctx)
	}
	if err := form(
		huh.NewNote().Title("Persona ID").Description(persona.ID),
		huh.NewInput().Title("Persona name").Value(&persona.Name),
		huh.NewInput().Title("GitHub host").Value(&persona.Host),
		huh.NewInput().Title("GitHub account").Value(&persona.Account),
		huh.NewInput().Title("Git author name").Value(&persona.GitName),
		huh.NewInput().Title("Git author email").Value(&persona.GitEmail),
	); err != nil {
		return err
	}
	if err := persona.Validate(); err != nil {
		return err
	}
	confirmed := false
	if err := form(huh.NewConfirm().Title("Save persona changes?").Value(&confirmed)); err != nil {
		return err
	}
	if !confirmed {
		return nil
	}
	_, err = client.Identity(ctx, identity.Request{Action: "persona.put", Persona: &persona})
	return err
}
