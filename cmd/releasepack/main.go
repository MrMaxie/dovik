// releasepack assembles and verifies native Dovik release archives.
package main

import (
	"archive/tar"
	"archive/zip"
	"bufio"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"time"

	"github.com/MrMaxie/dovik/internal/buildinfo"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type target struct {
	name      string
	archive   string
	exeSuffix string
}

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "releasepack:", err)
		os.Exit(1)
	}
}

func run(arguments []string) error {
	if len(arguments) == 0 {
		return errors.New("use package, verify, smoke, or check-licenses")
	}
	switch arguments[0] {
	case "package":
		flags := flag.NewFlagSet("package", flag.ContinueOnError)
		output := flags.String("output", "dist", "artifact output directory")
		if err := flags.Parse(arguments[1:]); err != nil {
			return err
		}
		return packageCurrent(*output)
	case "verify":
		flags := flag.NewFlagSet("verify", flag.ContinueOnError)
		archive := flags.String("archive", "", "release archive path")
		if err := flags.Parse(arguments[1:]); err != nil {
			return err
		}
		if *archive == "" {
			return errors.New("--archive is required")
		}
		return verifyArchive(*archive)
	case "smoke":
		flags := flag.NewFlagSet("smoke", flag.ContinueOnError)
		archive := flags.String("archive", "", "release archive path")
		if err := flags.Parse(arguments[1:]); err != nil {
			return err
		}
		if *archive == "" {
			return errors.New("--archive is required")
		}
		return smokeArchive(*archive)
	case "check-licenses":
		return checkLicenses()
	case "scoop":
		flags := flag.NewFlagSet("scoop", flag.ContinueOnError)
		archive := flags.String("archive", "", "Windows release archive path")
		output := flags.String("output", "", "rendered manifest path")
		if err := flags.Parse(arguments[1:]); err != nil {
			return err
		}
		if *archive == "" || *output == "" {
			return errors.New("--archive and --output are required")
		}
		return renderScoopManifest(*archive, *output)
	default:
		return fmt.Errorf("unknown command %q", arguments[0])
	}
}

func renderScoopManifest(archive, output string) error {
	if filepath.Base(archive) != "dovik-v"+buildinfo.Version+"-windows-x64.zip" {
		return errors.New("Scoop manifest requires the matching Windows x64 archive")
	}
	if err := verifyChecksum(archive); err != nil {
		return err
	}
	checksum, err := os.ReadFile(archive + ".sha256")
	if err != nil {
		return err
	}
	fields := strings.Fields(string(checksum))
	if len(fields) != 2 {
		return errors.New("invalid checksum file")
	}
	root, err := repositoryRoot()
	if err != nil {
		return err
	}
	template, err := os.ReadFile(filepath.Join(root, "packaging", "scoop", "dovik.json"))
	if err != nil {
		return err
	}
	manifest := strings.Replace(string(template), "RELEASE_SHA256_REQUIRED", strings.ToLower(fields[0]), 1)
	if manifest == string(template) {
		return errors.New("Scoop manifest template does not contain the release hash placeholder")
	}
	if err := os.MkdirAll(filepath.Dir(output), 0o755); err != nil {
		return err
	}
	return os.WriteFile(output, []byte(manifest), 0o644)
}

func currentTarget() (target, error) {
	key := runtime.GOOS + "/" + runtime.GOARCH
	targets := map[string]target{
		"windows/amd64": {name: "windows-x64", archive: ".zip", exeSuffix: ".exe"},
		"linux/amd64":   {name: "linux-x64", archive: ".tar.gz"},
		"darwin/amd64":  {name: "macos-x64", archive: ".tar.gz"},
		"darwin/arm64":  {name: "macos-arm64", archive: ".tar.gz"},
	}
	value, ok := targets[key]
	if !ok {
		return target{}, fmt.Errorf("unsupported native target %s", key)
	}
	return value, nil
}

func packageCurrent(output string) error {
	target, err := currentTarget()
	if err != nil {
		return err
	}
	root, err := repositoryRoot()
	if err != nil {
		return err
	}
	if err := checkLicensesAt(root); err != nil {
		return err
	}
	output, err = filepath.Abs(output)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(output, 0o755); err != nil {
		return err
	}
	temporary, err := os.MkdirTemp("", "dovik-release-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(temporary)

	base := fmt.Sprintf("dovik-v%s-%s", buildinfo.Version, target.name)
	content := filepath.Join(temporary, base)
	if err := os.MkdirAll(filepath.Join(content, "skills"), 0o755); err != nil {
		return err
	}
	for _, binary := range []string{"dovik", "dovikd", "gh"} {
		destination := filepath.Join(content, binary+target.exeSuffix)
		command := exec.Command("go", "build", "-trimpath", "-ldflags=-s -w", "-o", destination, "./cmd/"+binary)
		command.Dir = root
		command.Stdout = os.Stdout
		command.Stderr = os.Stderr
		if err := command.Run(); err != nil {
			return fmt.Errorf("build %s: %w", binary, err)
		}
		if runtime.GOOS != "windows" {
			if err := os.Chmod(destination, 0o755); err != nil {
				return err
			}
		}
	}
	for _, name := range []string{"README.md", "CHANGELOG.md", "LICENSE", "THIRD_PARTY_LICENSES.md"} {
		if err := copyFile(filepath.Join(root, name), filepath.Join(content, name), 0o644); err != nil {
			return err
		}
	}
	if err := copyTree(filepath.Join(root, "skills"), filepath.Join(content, "skills")); err != nil {
		return err
	}

	archive := filepath.Join(output, base+target.archive)
	if target.archive == ".zip" {
		err = writeZip(archive, temporary, base)
	} else {
		err = writeTarGzip(archive, temporary, base)
	}
	if err != nil {
		return err
	}
	if err := writeChecksum(archive); err != nil {
		return err
	}
	if err := verifyArchive(archive); err != nil {
		return err
	}
	fmt.Println(archive)
	return nil
}

func repositoryRoot() (string, error) {
	directory, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for {
		if _, err := os.Stat(filepath.Join(directory, "go.mod")); err == nil {
			return directory, nil
		}
		parent := filepath.Dir(directory)
		if parent == directory {
			return "", errors.New("could not find repository root")
		}
		directory = parent
	}
}

func copyFile(source, destination string, mode fs.FileMode) error {
	input, err := os.Open(source)
	if err != nil {
		return err
	}
	defer input.Close()
	if err := os.MkdirAll(filepath.Dir(destination), 0o755); err != nil {
		return err
	}
	output, err := os.OpenFile(destination, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, mode)
	if err != nil {
		return err
	}
	_, copyErr := io.Copy(output, input)
	closeErr := output.Close()
	return errors.Join(copyErr, closeErr)
}

func copyTree(source, destination string) error {
	return filepath.WalkDir(source, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		relative, err := filepath.Rel(source, path)
		if err != nil {
			return err
		}
		targetPath := filepath.Join(destination, relative)
		if entry.IsDir() {
			return os.MkdirAll(targetPath, 0o755)
		}
		return copyFile(path, targetPath, 0o644)
	})
}

func writeZip(destination, root, base string) error {
	file, err := os.Create(destination)
	if err != nil {
		return err
	}
	writer := zip.NewWriter(file)
	walkErr := filepath.Walk(filepath.Join(root, base), func(path string, info fs.FileInfo, walkErr error) error {
		if walkErr != nil || info.IsDir() {
			return walkErr
		}
		relative, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		header, err := zip.FileInfoHeader(info)
		if err != nil {
			return err
		}
		header.Name = filepath.ToSlash(relative)
		header.Method = zip.Deflate
		entry, err := writer.CreateHeader(header)
		if err != nil {
			return err
		}
		input, err := os.Open(path)
		if err != nil {
			return err
		}
		_, copyErr := io.Copy(entry, input)
		closeErr := input.Close()
		return errors.Join(copyErr, closeErr)
	})
	return errors.Join(walkErr, writer.Close(), file.Close())
}

func writeTarGzip(destination, root, base string) error {
	file, err := os.Create(destination)
	if err != nil {
		return err
	}
	gzipWriter := gzip.NewWriter(file)
	tarWriter := tar.NewWriter(gzipWriter)
	walkErr := filepath.Walk(filepath.Join(root, base), func(path string, info fs.FileInfo, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		relative, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		header, err := tar.FileInfoHeader(info, "")
		if err != nil {
			return err
		}
		header.Name = filepath.ToSlash(relative)
		header.ModTime = time.Unix(0, 0).UTC()
		if err := tarWriter.WriteHeader(header); err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		input, err := os.Open(path)
		if err != nil {
			return err
		}
		_, copyErr := io.Copy(tarWriter, input)
		closeErr := input.Close()
		return errors.Join(copyErr, closeErr)
	})
	return errors.Join(walkErr, tarWriter.Close(), gzipWriter.Close(), file.Close())
}

func writeChecksum(path string) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	hash := sha256.New()
	_, copyErr := io.Copy(hash, file)
	closeErr := file.Close()
	if err := errors.Join(copyErr, closeErr); err != nil {
		return err
	}
	line := fmt.Sprintf("%s  %s\n", hex.EncodeToString(hash.Sum(nil)), filepath.Base(path))
	return os.WriteFile(path+".sha256", []byte(line), 0o644)
}

func verifyArchive(archive string) error {
	archive, err := filepath.Abs(archive)
	if err != nil {
		return err
	}
	if err := verifyChecksum(archive); err != nil {
		return err
	}
	temporary, err := os.MkdirTemp("", "dovik-verify-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(temporary)
	if strings.HasSuffix(archive, ".zip") {
		err = extractZip(archive, temporary)
	} else if strings.HasSuffix(archive, ".tar.gz") {
		err = extractTarGzip(archive, temporary)
	} else {
		return errors.New("archive must end in .zip or .tar.gz")
	}
	if err != nil {
		return err
	}
	entries, err := archiveFiles(temporary)
	if err != nil {
		return err
	}
	base := strings.TrimSuffix(filepath.Base(archive), ".zip")
	base = strings.TrimSuffix(base, ".tar.gz")
	suffix := ""
	if strings.Contains(base, "-windows-") {
		suffix = ".exe"
	}
	expected := []string{
		base + "/CHANGELOG.md",
		base + "/LICENSE",
		base + "/README.md",
		base + "/THIRD_PARTY_LICENSES.md",
		base + "/dovik" + suffix,
		base + "/dovikd" + suffix,
		base + "/gh" + suffix,
		base + "/skills/README.md",
		base + "/skills/claude/SKILL.md",
		base + "/skills/codex/SKILL.md",
	}
	sort.Strings(expected)
	if strings.Join(entries, "\n") != strings.Join(expected, "\n") {
		return fmt.Errorf("archive file set differs\nwant:\n%s\ngot:\n%s", strings.Join(expected, "\n"), strings.Join(entries, "\n"))
	}
	content := filepath.Join(temporary, base)
	for _, binary := range []string{"dovik", "dovikd"} {
		path := filepath.Join(content, binary+suffix)
		if runtime.GOOS != "windows" {
			info, err := os.Stat(path)
			if err != nil || info.Mode()&0o111 == 0 {
				return fmt.Errorf("%s is not executable", binary)
			}
		}
		if err := expectOutput(path, "--version", binary+" "+buildinfo.Version); err != nil {
			return err
		}
		if err := expectSuccess(path, "--help"); err != nil {
			return err
		}
	}
	fmt.Println("verified", filepath.Base(archive))
	return nil
}

func archiveFiles(root string) ([]string, error) {
	var files []string
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil || entry.IsDir() {
			return walkErr
		}
		relative, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		files = append(files, filepath.ToSlash(relative))
		return nil
	})
	sort.Strings(files)
	return files, err
}

func verifyChecksum(archive string) error {
	line, err := os.ReadFile(archive + ".sha256")
	if err != nil {
		return err
	}
	fields := strings.Fields(string(line))
	if len(fields) != 2 || fields[1] != filepath.Base(archive) {
		return errors.New("invalid checksum file")
	}
	file, err := os.Open(archive)
	if err != nil {
		return err
	}
	hash := sha256.New()
	_, copyErr := io.Copy(hash, file)
	closeErr := file.Close()
	if err := errors.Join(copyErr, closeErr); err != nil {
		return err
	}
	if hex.EncodeToString(hash.Sum(nil)) != strings.ToLower(fields[0]) {
		return errors.New("SHA-256 mismatch")
	}
	return nil
}

func extractZip(archive, destination string) error {
	reader, err := zip.OpenReader(archive)
	if err != nil {
		return err
	}
	defer reader.Close()
	for _, entry := range reader.File {
		path, err := safeExtractPath(destination, entry.Name)
		if err != nil {
			return err
		}
		if entry.FileInfo().IsDir() {
			if err := os.MkdirAll(path, 0o755); err != nil {
				return err
			}
			continue
		}
		input, err := entry.Open()
		if err != nil {
			return err
		}
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			input.Close()
			return err
		}
		output, err := os.OpenFile(path, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, entry.Mode())
		if err != nil {
			input.Close()
			return err
		}
		copyErr := func() error {
			_, err := io.Copy(output, input)
			return errors.Join(err, output.Close(), input.Close())
		}()
		if copyErr != nil {
			return copyErr
		}
	}
	return nil
}

func extractTarGzip(archive, destination string) error {
	file, err := os.Open(archive)
	if err != nil {
		return err
	}
	defer file.Close()
	gzipReader, err := gzip.NewReader(file)
	if err != nil {
		return err
	}
	defer gzipReader.Close()
	reader := tar.NewReader(gzipReader)
	for {
		header, err := reader.Next()
		if errors.Is(err, io.EOF) {
			return nil
		}
		if err != nil {
			return err
		}
		path, err := safeExtractPath(destination, header.Name)
		if err != nil {
			return err
		}
		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(path, fs.FileMode(header.Mode)); err != nil {
				return err
			}
		case tar.TypeReg:
			if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
				return err
			}
			output, err := os.OpenFile(path, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, fs.FileMode(header.Mode))
			if err != nil {
				return err
			}
			_, copyErr := io.Copy(output, reader)
			if err := errors.Join(copyErr, output.Close()); err != nil {
				return err
			}
		default:
			return fmt.Errorf("unsupported archive entry %q", header.Name)
		}
	}
}

func safeExtractPath(root, name string) (string, error) {
	path := filepath.Join(root, filepath.FromSlash(name))
	cleanRoot := filepath.Clean(root) + string(os.PathSeparator)
	if !strings.HasPrefix(filepath.Clean(path)+string(os.PathSeparator), cleanRoot) {
		return "", fmt.Errorf("unsafe archive path %q", name)
	}
	return path, nil
}

func expectOutput(path, argument, expected string) error {
	output, err := exec.Command(path, argument).CombinedOutput()
	if err != nil {
		return fmt.Errorf("%s %s failed: %w: %s", filepath.Base(path), argument, err, output)
	}
	if strings.TrimSpace(string(output)) != expected {
		return fmt.Errorf("%s %s output = %q, want %q", filepath.Base(path), argument, strings.TrimSpace(string(output)), expected)
	}
	return nil
}

func expectSuccess(path string, arguments ...string) error {
	if output, err := exec.Command(path, arguments...).CombinedOutput(); err != nil {
		return fmt.Errorf("%s %s failed: %w: %s", filepath.Base(path), strings.Join(arguments, " "), err, output)
	}
	return nil
}

func smokeArchive(archive string) error {
	temporary, err := os.MkdirTemp("", "dovik-smoke-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(temporary)
	if strings.HasSuffix(archive, ".zip") {
		err = extractZip(archive, temporary)
	} else {
		err = extractTarGzip(archive, temporary)
	}
	if err != nil {
		return err
	}
	base := strings.TrimSuffix(filepath.Base(archive), ".zip")
	base = strings.TrimSuffix(base, ".tar.gz")
	suffix := ""
	if runtime.GOOS == "windows" {
		suffix = ".exe"
	}
	content := filepath.Join(temporary, base)
	dovik := filepath.Join(content, "dovik"+suffix)
	daemon := filepath.Join(content, "dovikd"+suffix)
	if err := expectFailureContaining(exec.Command(dovik, "tui"), "interactive"); err != nil {
		return fmt.Errorf("TUI boundary: %w", err)
	}

	state := filepath.Join(temporary, "state")
	runtimeDirectory := filepath.Join(temporary, "runtime")
	if err := os.MkdirAll(runtimeDirectory, 0o700); err != nil {
		return err
	}
	environment := append(os.Environ(), "LOCALAPPDATA="+state, "XDG_STATE_HOME="+state, "XDG_RUNTIME_DIR="+runtimeDirectory)
	preflight := exec.Command(dovik, "--json", "project", "list")
	preflight.Env = environment
	if preflight.Run() == nil {
		return errors.New("a Dovik daemon is already reachable; isolated smoke test refused")
	}
	command := exec.Command(daemon)
	command.Env = environment
	var daemonOutput strings.Builder
	command.Stdout = &daemonOutput
	command.Stderr = &daemonOutput
	if err := command.Start(); err != nil {
		return err
	}
	defer func() {
		_ = command.Process.Kill()
		_ = command.Wait()
	}()

	deadline := time.Now().Add(10 * time.Second)
	for {
		probe := exec.Command(dovik, "--json", "project", "list")
		probe.Env = command.Env
		if err := probe.Run(); err == nil {
			break
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("daemon did not become ready: %s", daemonOutput.String())
		}
		time.Sleep(100 * time.Millisecond)
	}
	projectRoot := filepath.Join(temporary, "project")
	if err := os.MkdirAll(projectRoot, 0o755); err != nil {
		return err
	}
	gitInit := exec.Command("git", "init", "--quiet", projectRoot)
	gitInit.Env = command.Env
	if output, err := gitInit.CombinedOutput(); err != nil {
		return fmt.Errorf("initialize smoke repository: %w: %s", err, output)
	}
	if err := runWithEnvironment(command.Env, dovik, "--json", "project", "add", "--id", "release-smoke", "--root", projectRoot); err != nil {
		return err
	}
	if err := runWithEnvironment(command.Env, dovik, "--json", "process", "add", "--project", "release-smoke", "--id", "worker", "--command", dovik, "--arg", "agent-idle"); err != nil {
		return err
	}
	if err := smokeMCP(command.Env, dovik); err != nil {
		return err
	}
	proxy := withEnvironment(command.Env, exec.Command(filepath.Join(content, "gh"+suffix), "--dovik-proxy-identify"))
	output, err := proxy.CombinedOutput()
	if err != nil {
		return fmt.Errorf("gh proxy boundary: %w: %s", err, output)
	}
	if strings.TrimSpace(string(output)) != "dovik-gh-proxy-v1" {
		return fmt.Errorf("gh proxy boundary returned unexpected output: %s", output)
	}
	fmt.Println("smoked", filepath.Base(archive))
	return nil
}

func smokeMCP(environment []string, dovik string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	command := exec.CommandContext(ctx, dovik, "mcp")
	command.Env = environment
	session, err := mcp.NewClient(&mcp.Implementation{Name: "release-smoke", Version: buildinfo.Version}, nil).Connect(ctx, &mcp.CommandTransport{Command: command}, nil)
	if err != nil {
		return fmt.Errorf("initialize MCP smoke session: %w", err)
	}
	defer session.Close()
	tools, err := session.ListTools(ctx, nil)
	if err != nil {
		return fmt.Errorf("discover MCP tools: %w", err)
	}
	if len(tools.Tools) != 7 {
		return fmt.Errorf("discover seven MCP tools: count=%d", len(tools.Tools))
	}
	process := map[string]any{"projectId": "release-smoke", "processId": "worker"}
	calls := []struct {
		name      string
		arguments any
	}{
		{name: "list_projects", arguments: map[string]any{}},
		{name: "list_processes", arguments: map[string]any{"projectId": "release-smoke"}},
		{name: "process_status", arguments: process},
		{name: "process_start", arguments: process},
		{name: "process_logs", arguments: process},
		{name: "process_restart", arguments: process},
		{name: "process_stop", arguments: process},
	}
	for _, call := range calls {
		result, err := session.CallTool(ctx, &mcp.CallToolParams{Name: call.name, Arguments: call.arguments})
		if err != nil {
			return fmt.Errorf("MCP %s: %w", call.name, err)
		}
		if result.IsError {
			return fmt.Errorf("MCP %s returned an application error", call.name)
		}
	}
	if err := session.Close(); err != nil {
		return fmt.Errorf("close MCP smoke session: %w", err)
	}
	return nil
}

func withEnvironment(environment []string, command *exec.Cmd) *exec.Cmd {
	command.Env = environment
	return command
}

func runWithEnvironment(environment []string, path string, arguments ...string) error {
	command := exec.Command(path, arguments...)
	command.Env = environment
	if output, err := command.CombinedOutput(); err != nil {
		return fmt.Errorf("%s failed: %w: %s", strings.Join(arguments, " "), err, output)
	}
	return nil
}

func expectFailureContaining(command *exec.Cmd, expected string) error {
	output, err := command.CombinedOutput()
	if err == nil {
		return fmt.Errorf("command unexpectedly succeeded: %s", output)
	}
	if !strings.Contains(strings.ToLower(string(output)), strings.ToLower(expected)) {
		return fmt.Errorf("output %q does not contain %q", output, expected)
	}
	return nil
}

func checkLicenses() error {
	root, err := repositoryRoot()
	if err != nil {
		return err
	}
	return checkLicensesAt(root)
}

func checkLicensesAt(root string) error {
	command := exec.Command("go", "list", "-deps", "-f", `{{with .Module}}{{if .Version}}{{.Path}}|{{.Version}}|{{.Dir}}{{end}}{{end}}`, "./cmd/dovik", "./cmd/dovikd", "./cmd/gh")
	command.Dir = root
	output, err := command.Output()
	if err != nil {
		return fmt.Errorf("list runtime modules: %w", err)
	}
	missing := []string{}
	seen := map[string]bool{}
	scanner := bufio.NewScanner(strings.NewReader(string(output)))
	for scanner.Scan() {
		fields := strings.Split(scanner.Text(), "|")
		if len(fields) != 3 || seen[fields[0]] {
			continue
		}
		seen[fields[0]] = true
		found := false
		for _, name := range []string{"LICENSE", "LICENSE.md", "LICENSE.txt", "COPYING", "NOTICE"} {
			if info, err := os.Stat(filepath.Join(fields[2], name)); err == nil && !info.IsDir() {
				found = true
				break
			}
		}
		if !found {
			missing = append(missing, fields[0]+"@"+fields[1])
		}
	}
	if err := scanner.Err(); err != nil {
		return err
	}
	if len(missing) != 0 {
		sort.Strings(missing)
		return fmt.Errorf("runtime modules without a root license file: %s", strings.Join(missing, ", "))
	}
	return nil
}
