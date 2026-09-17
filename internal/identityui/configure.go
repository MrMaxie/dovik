// Package identityui provides the shared operator questionnaire for CLI and TUI.
package identityui

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"

	"charm.land/huh/v2"
	"github.com/MrMaxie/dovik/internal/identity"
	"github.com/MrMaxie/dovik/internal/operatorclient"
	"github.com/MrMaxie/dovik/internal/supervision"
	"github.com/MrMaxie/dovik/internal/terminalstyle"
)

func Configure(ctx context.Context, client operatorclient.Client, root string, input io.Reader, output io.Writer) error {
	err := configure(ctx, client, root, input, output)
	if errors.Is(err, huh.ErrUserAborted) {
		return nil
	}
	return err
}

func configure(ctx context.Context, client operatorclient.Client, root string, input io.Reader, output io.Writer) error {
	root, err := identity.GitRoot(ctx, root)
	if err != nil {
		return err
	}
	snapshot, err := client.Identity(ctx, identity.Request{Action: "get"})
	if err != nil {
		return err
	}
	project := identity.Project{
		ID:           filepath.Base(root),
		Name:         filepath.Base(root),
		Root:         root,
		Mode:         "proxy-level",
		ProxyEnabled: true,
		Alternatives: []string{},
		Policy:       identity.Policy{Preset: "read-only", Exceptions: map[string]bool{}},
	}
	if existing, e := snapshot.State.FindProject("", root); e == nil {
		project = existing
	}
	defaults := identity.DiscoverGitDefaults(ctx, root)
	if project.Repository == "" {
		project.Repository = defaults.Repository
	}
	gh := snapshot.State.GHPath
	if gh == "" {
		gh, _ = exec.LookPath("realgh")
		if gh == "" {
			gh, _ = exec.LookPath("gh")
		}
	}
	form := func(fields ...huh.Field) error {
		return terminalstyle.NewForm("Configure project identity", input, output, fields...).RunWithContext(ctx)
	}
	fields := projectDetailsFields(&project, &gh, snapshot.State.GHPath == "")
	if err := form(fields...); err != nil {
		return err
	}
	if _, err := identity.ValidateGH(gh); err != nil {
		return err
	}
	accounts, discoveryErr := identity.GitHubAccounts(ctx, gh)
	if discoveryErr != nil {
		fmt.Fprintln(output, "GitHub CLI accounts could not be listed. Select a saved persona or create one manually.")
	}
	personas := personaChoices(snapshot.State.Personas, accounts, defaults)
	options := []huh.Option[string]{}
	for _, p := range personas {
		options = append(options, huh.NewOption(p.Name+" ("+p.Host+"/"+p.Account+")", p.ID))
	}
	options = append(options, huh.NewOption("Create a persona", "__new"))
	chosen := project.Persona
	if chosen == "" && len(options) > 0 {
		chosen = options[0].Value
	}
	if err := form(huh.NewSelect[string]().Title("Who are you in this project?").Options(options...).Value(&chosen)); err != nil {
		return err
	}
	persona := identity.Persona{Host: "github.com", GitName: defaults.Name, GitEmail: defaults.Email}
	if defaults.Host != "" {
		persona.Host = defaults.Host
	}
	if chosen != "__new" {
		for _, p := range personas {
			if p.ID == chosen {
				persona = p
				break
			}
		}
	}
	_, savedErr := snapshot.State.FindPersona(chosen)
	if chosen == "__new" || savedErr != nil {
		if err := form(huh.NewInput().Title("Persona ID").Value(&persona.ID), huh.NewInput().Title("Persona name").Value(&persona.Name), huh.NewInput().Title("GitHub host").Value(&persona.Host), huh.NewInput().Title("GitHub account").Value(&persona.Account), huh.NewInput().Title("Git author name").Value(&persona.GitName), huh.NewInput().Title("Git author email").Value(&persona.GitEmail)); err != nil {
			return err
		}
	} else {
		persona, err = snapshot.State.FindPersona(chosen)
		if err != nil {
			return err
		}
	}
	if err := persona.Validate(); err != nil {
		return err
	}
	project.Persona = persona.ID
	applyGit := false
	if err := form(
		huh.NewConfirm().Title("Set this persona as the repository-local Git author?").Description(persona.GitName+" <"+persona.GitEmail+">").Value(&applyGit),
		huh.NewSelect[string]().Title("Agent protection").Options(
			huh.NewOption("Proxy-level: transparently route the selected persona", "proxy-level"),
			huh.NewOption("Agent isolation: enforce policy in a container or separate account", "agent-isolation"),
		).Value(&project.Mode),
	); err != nil {
		return err
	}
	allowed := []string{}
	if requiresIsolationPolicy(project.Mode) {
		if err := form(
			huh.NewSelect[string]().Title("Choose the agent's starting permissions").Options(
				huh.NewOption("Read only", "read-only"),
				huh.NewOption("Collaborate on issues and pull requests", "collaborate"),
				huh.NewOption("Maintain this repository", "maintain"),
			).Value(&project.Policy.Preset),
		); err != nil {
			return err
		}
		permissionOptions := []huh.Option[string]{}
		labels := map[string]string{"read": "Read repository, issues, PRs and Actions", "comment": "Post comments", "create": "Create issues and PRs", "edit": "Edit issues and PRs; mark PR ready", "close": "Close and reopen issues and PRs", "review": "Submit PR reviews", "merge": "Merge PRs", "actions": "Run workflows, rerun or cancel Actions", "persona": "Select an approved alternative persona"}
		for _, permission := range identity.Permissions {
			permissionOptions = append(permissionOptions, huh.NewOption(labels[permission], permission))
			if project.Policy.Allows(permission) {
				allowed = append(allowed, permission)
			}
		}
		if err := form(huh.NewMultiSelect[string]().Title("What may the agent do?").Description("Unchecked operations are denied. Credential export and policy administration are never granted.").Options(permissionOptions...).Value(&allowed)); err != nil {
			return err
		}
		project.Policy.Exceptions = map[string]bool{}
		for _, permission := range identity.Permissions {
			project.Policy.Exceptions[permission] = slices.Contains(allowed, permission)
		}
		if slices.Contains(allowed, "persona") {
			alternatives := []huh.Option[string]{}
			for _, p := range snapshot.State.Personas {
				if p.ID != persona.ID && p.Host == persona.Host {
					alternatives = append(alternatives, huh.NewOption(p.Name, p.ID))
				}
			}
			if len(alternatives) > 0 {
				if err := form(huh.NewMultiSelect[string]().Title("Which other personas may the agent select?").Options(alternatives...).Value(&project.Alternatives)); err != nil {
					return err
				}
			}
		}
	}
	confirmed := false
	projects, err := client.ListProjects(ctx)
	if err != nil {
		return err
	}
	found := false
	for _, p := range projects {
		if string(p.ID) == project.ID {
			found = true
			if p.RootDirectory != project.Root {
				return fmt.Errorf("the process registry project has a different root; choose another project ID")
			}
		}
	}
	summary := configurationSummary(project, persona, allowed, applyGit)
	if err := form(huh.NewConfirm().Title("Save this project configuration?").Description(summary).Value(&confirmed)); err != nil {
		return err
	}
	if !confirmed {
		return nil
	}
	if _, err := client.Identity(ctx, identity.Request{Action: "configure", Persona: &persona, Project: &project, Path: gh}); err != nil {
		return err
	}
	if !found {
		if _, err := client.AddProject(ctx, supervision.ProjectDefinition{ID: supervision.ProjectID(project.ID), RootDirectory: project.Root}); err != nil {
			return err
		}
	}
	if err := identity.ApplyGitCredentialHelper(ctx, root); err != nil {
		return err
	}
	if applyGit {
		return identity.ApplyGitAuthor(ctx, root, persona)
	}
	return nil
}

func requiresIsolationPolicy(mode string) bool {
	return mode == "agent-isolation"
}

func configurationSummary(project identity.Project, persona identity.Persona, allowed []string, applyGit bool) string {
	summary := fmt.Sprintf("%s\n%s\n%s/%s\nPersona: %s (%s)\nProtection: %s", project.Name, project.Description, persona.Host, project.Repository, persona.Name, persona.Account, project.Mode)
	if requiresIsolationPolicy(project.Mode) {
		summary += fmt.Sprintf("\nAllowed: %v", allowed)
	}
	return summary + fmt.Sprintf("\nRoute this repository's Git credentials through gh: yes\nUpdate local Git author: %t", applyGit)
}

func projectDetailsFields(project *identity.Project, gh *string, requestGHPath bool) []huh.Field {
	fields := []huh.Field{
		huh.NewInput().Title("Project ID").Value(&project.ID),
		huh.NewInput().Title("Project name").Value(&project.Name),
		huh.NewText().Title("What is this project?").CharLimit(4096).Value(&project.Description),
	}
	if project.Repository == "" {
		fields = append(fields, huh.NewInput().Title("GitHub repository (OWNER/REPO)").Value(&project.Repository))
	} else {
		fields = append(fields, huh.NewNote().Title("GitHub repository").Description(project.Repository))
	}
	if requestGHPath {
		fields = append(fields, huh.NewInput().Key("gh-path").Title("Original gh executable (absolute path)").Value(gh).Validate(func(v string) error {
			if !filepath.IsAbs(v) {
				return fmt.Errorf("use an absolute executable path")
			}
			return nil
		}))
	}
	return fields
}

// Discovered accounts are suggestions only; Configure saves the selected persona
// together with the project after the final confirmation.
func personaChoices(saved []identity.Persona, accounts []identity.GitHubAccount, defaults identity.GitDefaults) []identity.Persona {
	result := append([]identity.Persona{}, saved...)
	ids := map[string]bool{}
	for _, p := range saved {
		ids[p.ID] = true
	}
	for _, account := range accounts {
		if defaults.Host != "" && !strings.EqualFold(defaults.Host, account.Host) {
			continue
		}
		found := false
		for _, p := range saved {
			if strings.EqualFold(p.Host, account.Host) && strings.EqualFold(p.Account, account.Account) {
				found = true
				break
			}
		}
		if found {
			continue
		}
		id := account.Account
		for suffix := 2; ids[id]; suffix++ {
			id = fmt.Sprintf("%s-%d", account.Account, suffix)
		}
		ids[id] = true
		result = append(result, identity.Persona{ID: id, Name: account.Account, Host: account.Host, Account: account.Account, GitName: defaults.Name, GitEmail: defaults.Email})
	}
	return result
}
