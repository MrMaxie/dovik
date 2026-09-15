package identity

import (
	"fmt"
	"regexp"
	"slices"
	"strconv"
	"strings"
)

type Invocation struct {
	Arguments []string `json:"arguments"`
	Input     []byte   `json:"input,omitempty"`
}

type Command struct {
	Arguments   []string
	Permission  string
	Credential  bool
	Interactive bool
}

type commandSpec struct {
	permission, values, booleans string
	positional                   bool
}

var catalog = map[string]commandSpec{
	"repo view":     {"read", "branch", "", false},
	"pr list":       {"read", "state,base,head,author,assignee,label,limit,search", "draft", false},
	"pr view":       {"read", "", "comments", true},
	"pr diff":       {"read", "color", "patch,name-only", true},
	"pr checks":     {"read", "", "required", true},
	"pr create":     {"create", "title,body,body-file,head,base,assignee,label,reviewer,milestone", "draft,no-maintainer-edit", false},
	"pr comment":    {"comment", "body,body-file", "", true},
	"pr edit":       {"edit", "title,body,body-file,base,add-label,remove-label,add-assignee,remove-assignee,add-reviewer,remove-reviewer,milestone", "", true},
	"pr close":      {"close", "comment", "", true},
	"pr reopen":     {"close", "comment", "", true},
	"pr review":     {"review", "body,body-file", "approve,request-changes,comment", true},
	"pr ready":      {"edit", "", "undo", true},
	"pr merge":      {"merge", "match-head-commit", "merge,squash,rebase,auto,disable-auto", true},
	"issue list":    {"read", "state,author,assignee,label,limit,search", "", false},
	"issue view":    {"read", "", "comments", true},
	"issue create":  {"create", "title,body,body-file,assignee,label,milestone", "", false},
	"issue comment": {"comment", "body,body-file", "", true},
	"issue edit":    {"edit", "title,body,body-file,add-label,remove-label,add-assignee,remove-assignee,milestone", "", true},
	"issue close":   {"close", "comment,reason", "", true},
	"issue reopen":  {"close", "comment", "", true},
	"run list":      {"read", "workflow,branch,event,status,user,limit,commit,created", "", false},
	"run view":      {"read", "job,attempt", "log,log-failed,exit-status", true},
	"run rerun":     {"actions", "job", "failed", true},
	"run cancel":    {"actions", "", "", true},
	"workflow list": {"read", "limit", "all", false},
	"workflow view": {"read", "ref", "yaml", true},
	"workflow run":  {"actions", "ref,raw-field", "", true},
}

var shortFlags = map[string]string{"R": "repo", "b": "body", "F": "body-file", "f": "raw-field", "t": "title", "H": "head", "B": "base", "L": "limit", "q": "jq", "s": "state", "a": "assignee", "l": "label"}
var number = regexp.MustCompile(`^[1-9][0-9]*$`)
var workflowName = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9_.-]*$`)

// Authorize parses a closed command and flag catalogue before any credential access.
func Authorize(project Project, persona Persona, in Invocation) (Command, error) {
	deny := func() (Command, error) {
		return Command{}, fmt.Errorf("operation or arguments are not permitted by the project policy")
	}
	if len(in.Input) > 1024*1024 || len(in.Arguments) < 2 || len(in.Arguments) > 256 {
		return deny()
	}
	for _, arg := range in.Arguments {
		if len(arg) > 1024*1024 || strings.ContainsRune(arg, 0) {
			return deny()
		}
	}
	if isGitCredentialInvocation(in) {
		return authorizeGitCredential(project, persona, in)
	}
	if in.Arguments[0] == "api" {
		return authorizeAPI(project, persona, in)
	}
	name := strings.Join(in.Arguments[:2], " ")
	spec, ok := catalog[name]
	if !ok || !project.Policy.Allows(spec.permission) {
		return deny()
	}
	if name == "pr create" && len(in.Arguments) == 2 {
		return Command{
			Arguments:   []string{"pr", "create", "--repo", persona.Host + "/" + project.Repository},
			Permission:  spec.permission,
			Interactive: true,
		}, nil
	}
	values := map[string]string{}
	output := append([]string{}, in.Arguments[:2]...)
	positional := 0
	for i := 2; i < len(in.Arguments); i++ {
		arg := in.Arguments[i]
		if !strings.HasPrefix(arg, "-") {
			if !spec.positional || positional > 0 {
				return deny()
			}
			if strings.HasPrefix(name, "workflow ") {
				if !workflowName.MatchString(arg) {
					return deny()
				}
			} else if !number.MatchString(arg) {
				return deny()
			}
			output = append(output, arg)
			positional++
			continue
		}
		flag, value, hasValue := strings.Cut(strings.TrimPrefix(arg, "--"), "=")
		if !strings.HasPrefix(arg, "--") {
			flag = strings.TrimPrefix(arg, "-")
			flag = shortFlags[flag]
			if flag == "" {
				return deny()
			}
		}
		if flag == "" {
			return deny()
		}
		isValue := slices.Contains(strings.Split(spec.values, ","), flag) || flag == "repo" || (spec.permission == "read" && slices.Contains([]string{"json", "jq", "template"}, flag))
		isBool := slices.Contains(strings.Split(spec.booleans, ","), flag) && flag != ""
		if !isValue && !isBool {
			return deny()
		}
		if _, duplicate := values[flag]; duplicate && flag != "raw-field" {
			return deny()
		}
		if isBool {
			if hasValue {
				return deny()
			}
			values[flag] = "true"
			output = append(output, "--"+flag)
			continue
		}
		if !hasValue {
			i++
			if i == len(in.Arguments) {
				return deny()
			}
			value = in.Arguments[i]
		}
		values[flag] = value
		if flag == "repo" {
			if !sameRepository(value, project, persona) {
				return deny()
			}
			continue
		}
		if flag == "body-file" && value != "-" {
			return deny()
		}
		if flag == "limit" {
			n, e := strconv.Atoi(value)
			if e != nil || n < 1 || n > 1000 {
				return deny()
			}
		}
		output = append(output, "--"+flag, value)
	}
	if spec.positional && positional != 1 {
		return deny()
	}
	if spec.permission == "close" && values["comment"] != "" && !project.Policy.Allows("comment") {
		return deny()
	}
	if strings.HasSuffix(name, " create") {
		if values["title"] == "" || (values["body"] == "" && values["body-file"] != "-") {
			return deny()
		}
		if name == "pr create" && (values["head"] == "" || values["base"] == "") {
			return deny()
		}
	}
	if strings.HasSuffix(name, " comment") && values["body"] == "" && values["body-file"] != "-" {
		return deny()
	}
	if name == "pr review" {
		count := 0
		for _, f := range []string{"approve", "request-changes", "comment"} {
			if values[f] != "" {
				count++
			}
		}
		if count != 1 {
			return deny()
		}
	}
	output = append(output, "--repo", persona.Host+"/"+project.Repository)
	return Command{Arguments: output, Permission: spec.permission}, nil
}

func authorizeGitCredential(project Project, persona Persona, in Invocation) (Command, error) {
	deny := func() (Command, error) {
		return Command{}, fmt.Errorf("Git credential request is not valid for this project")
	}
	if len(in.Arguments) != 3 || in.Arguments[2] != "get" || len(in.Input) == 0 || len(in.Input) > 64*1024 {
		return deny()
	}
	fields := map[string]string{}
	seen := map[string]bool{}
	for _, line := range strings.Split(strings.ReplaceAll(string(in.Input), "\r\n", "\n"), "\n") {
		if line == "" {
			continue
		}
		key, value, ok := strings.Cut(line, "=")
		if !ok {
			return deny()
		}
		if key == "capability[]" {
			if !slices.Contains([]string{"authtype", "state"}, value) {
				return deny()
			}
			continue
		}
		if key == "wwwauth[]" {
			if value == "" || len(value) > 4096 {
				return deny()
			}
			continue
		}
		if seen[key] || !slices.Contains([]string{"protocol", "host", "path", "username"}, key) {
			return deny()
		}
		seen[key] = true
		fields[key] = value
	}
	if fields["protocol"] != "https" || !strings.EqualFold(fields["host"], persona.Host) {
		return deny()
	}
	if path := strings.TrimSuffix(strings.Trim(fields["path"], "/"), ".git"); path != "" && !strings.EqualFold(path, project.Repository) {
		return deny()
	}
	return Command{Credential: true}, nil
}

func isGitCredentialInvocation(in Invocation) bool {
	return len(in.Arguments) >= 2 && in.Arguments[0] == "auth" && in.Arguments[1] == "git-credential"
}

func sameRepository(value string, p Project, persona Persona) bool {
	return strings.EqualFold(value, p.Repository) || strings.EqualFold(value, persona.Host+"/"+p.Repository)
}

type apiRoute struct{ method, path, permission, fields string }

var apiRoutes = []apiRoute{
	{"GET", "", "read", ""},
	{"GET", "issues", "read", "state,labels,assignee,creator,page,per_page"},
	{"GET", "issues/#", "read", ""},
	{"GET", "issues/#/comments", "read", "page,per_page"},
	{"GET", "pulls", "read", "state,head,base,page,per_page"},
	{"GET", "pulls/#", "read", ""},
	{"GET", "pulls/#/reviews", "read", "page,per_page"},
	{"GET", "actions/runs", "read", "branch,event,status,page,per_page"},
	{"GET", "actions/runs/#", "read", ""},
	{"GET", "actions/workflows", "read", "page,per_page"},
	{"POST", "issues", "create", "title,body"},
	{"POST", "issues/#/comments", "comment", "body"},
	{"PATCH", "issues/#", "edit", "title,body"},
	{"POST", "pulls", "create", "title,body,head,base"},
	{"PATCH", "pulls/#", "edit", "title,body,base"},
	{"PUT", "pulls/#/merge", "merge", "merge_method,commit_title,commit_message,sha"},
	{"POST", "actions/runs/#/rerun", "actions", ""},
	{"POST", "actions/runs/#/cancel", "actions", ""},
	{"POST", "actions/workflows/#/dispatches", "actions", "ref"},
}

func authorizeAPI(project Project, persona Persona, in Invocation) (Command, error) {
	deny := func() (Command, error) {
		return Command{}, fmt.Errorf("REST operation is not in the permitted catalogue")
	}
	endpoint := strings.TrimPrefix(in.Arguments[1], "/")
	prefix := "repos/" + project.Repository
	if endpoint != prefix && !strings.HasPrefix(endpoint, prefix+"/") {
		return deny()
	}
	suffix := strings.TrimPrefix(strings.TrimPrefix(endpoint, prefix), "/")
	parts := strings.Split(suffix, "/")
	for i, p := range parts {
		if number.MatchString(p) {
			parts[i] = "#"
		}
	}
	routePath := strings.Join(parts, "/")
	method := "GET"
	methodSet := false
	fields := []string{}
	format := []string{}
	for i := 2; i < len(in.Arguments); i++ {
		flag := in.Arguments[i]
		i++
		if i == len(in.Arguments) {
			return deny()
		}
		value := in.Arguments[i]
		switch flag {
		case "--method", "-X":
			if methodSet {
				return deny()
			}
			methodSet = true
			method = value
		case "--raw-field", "-f":
			fields = append(fields, value)
		case "--jq", "--template":
			format = append(format, flag, value)
		default:
			return deny()
		}
	}
	for _, route := range apiRoutes {
		if route.method != method || route.path != routePath || !project.Policy.Allows(route.permission) {
			continue
		}
		seen := map[string]bool{}
		for _, field := range fields {
			key, _, ok := strings.Cut(field, "=")
			if !ok || seen[key] || !slices.Contains(strings.Split(route.fields, ","), key) {
				return deny()
			}
			seen[key] = true
		}
		args := []string{"api", endpoint, "--hostname", persona.Host, "--method", method}
		for _, f := range fields {
			args = append(args, "--raw-field", f)
		}
		args = append(args, format...)
		return Command{Arguments: args, Permission: route.permission}, nil
	}
	return deny()
}
