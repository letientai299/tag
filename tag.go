package main

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"slices"
	"text/template"

	"github.com/fatih/color"
)

func check(e error) {
	if e != nil {
		log.Fatal(e)
	}
}

func extractCmdExitCode(err error) int {
	if err == nil {
		return 0
	}
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		return exitErr.ExitCode()
	}
	return 1
}

func optionIndex(args []string, option string) int {
	for i := len(args) - 1; i >= 0; i-- {
		if args[i] == option {
			return i
		}
	}
	return -1
}

func isatty(f *os.File) bool {
	stat, err := f.Stat()
	check(err)
	return stat.Mode()&os.ModeCharDevice != 0
}

func getEnvDefault(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}

var (
	red          = color.RedString
	blue         = color.BlueString
	lineNumberRe = regexp.MustCompile(`^(\d+):(\d+):.*`)
	ansi         = regexp.MustCompile(`\x1b\[[0-9;]*[a-zA-Z]`) // Source: https://superuser.com/a/380778
)

type AliasFile struct {
	filename string
	tmpl     *template.Template
	buf      bytes.Buffer
}

func NewAliasFile() *AliasFile {
	aliasFilename := getEnvDefault("TAG_ALIAS_FILE", "/tmp/tag_aliases")
	aliasPrefix := getEnvDefault("TAG_ALIAS_PREFIX", "e")
	aliasCmdFmtString := getEnvDefault(
		"TAG_CMD_FMT_STRING",
		`vim -c "call cursor({{.LineNumber}}, {{.ColumnNumber}})" "{{.Filename}}"`)

	fmtStr := "alias " + aliasPrefix + "{{.MatchIndex}}='" + aliasCmdFmtString + "'\n"
	return &AliasFile{
		filename: aliasFilename,
		tmpl:     template.Must(template.New("alias").Parse(fmtStr)),
	}
}

func (a *AliasFile) WriteAlias(index int, filename, linenum string, colnum string) {
	aliasVars := struct {
		MatchIndex   int
		Filename     string
		LineNumber   string
		ColumnNumber string
	}{index, filename, linenum, colnum}

	err := a.tmpl.Execute(&a.buf, aliasVars)
	check(err)
}

func (a *AliasFile) WriteFile() {
	err := os.WriteFile(a.filename, a.buf.Bytes(), 0644)
	check(err)
}

func tagPrefix(aliasIndex int) string {
	return blue("[") + red("%d", aliasIndex) + blue("]")
}

func generateTags(cmd *exec.Cmd) int {
	cmd.Stderr = os.Stderr

	stdout, err := cmd.StdoutPipe()
	check(err)

	scanner := bufio.NewScanner(stdout)
	scanner.Split(bufio.ScanLines)

	var (
		line          []byte
		colorlessLine []byte
		curPath       string
		groupIdxs     []int
	)

	aliasFile := NewAliasFile()
	defer aliasFile.WriteFile()

	aliasIndex := 1

	err = cmd.Start()
	check(err)

	for scanner.Scan() {
		line = scanner.Bytes()
		colorlessLine = ansi.ReplaceAll(line, nil) // strip ANSI
		if len(curPath) == 0 {
			// Path is always in the first line of a group (the heading). Extract and print it
			curPath = string(colorlessLine)
			curPath, err = filepath.Abs(curPath)
			check(err)
			fmt.Println(string(line))
		} else if groupIdxs = lineNumberRe.FindSubmatchIndex(colorlessLine); len(groupIdxs) > 0 {
			// Extract and tag matches
			aliasFile.WriteAlias(
				aliasIndex,
				curPath,
				string(colorlessLine[groupIdxs[2]:groupIdxs[3]]),
				string(colorlessLine[groupIdxs[4]:groupIdxs[5]]))
			fmt.Printf("%s %s\n", tagPrefix(aliasIndex), string(line))
			aliasIndex++
		} else {
			// Empty line. End of grouping, reset curPath context
			fmt.Println(string(line))
			curPath = ""
		}
	}

	err = cmd.Wait()
	return extractCmdExitCode(err)
}

func passThrough(cmd *exec.Cmd) int {
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	err := cmd.Run()
	return extractCmdExitCode(err)
}

func constructTagArgs(userArgs []string) []string {
	if !isatty(os.Stdout) {
		return nil
	}
	// --with-filename forces the filename heading even when searching a single
	// file, which rg omits by default. Without it, tag treats the first match
	// line as the file path and generates broken aliases.
	base := []string{"--with-filename", "--heading", "--column"}
	// ripgrep can't handle more than one --color option, so if the user provides one
	// we have to explicilty keep tag from passing its own --color option
	if optionIndex(userArgs, "--color") >= 0 {
		return base
	}
	return append(base, "--color", "always")
}

func handleColorSetting(args []string) {
	colorFlagIdx := optionIndex(args, "--color")
	color.NoColor = colorFlagIdx >= 0 && colorFlagIdx+1 < len(args) && args[colorFlagIdx+1] == "never"
}

func main() {
	userArgs := os.Args[1:]

	disableTag := false
	var tagArgs []string

	switch i := optionIndex(userArgs, "--notag"); {
	case i > 0:
		userArgs = slices.Delete(userArgs, i, i+1)
		fallthrough
	case len(userArgs) == 0: // no arguments; fall back to help message
		disableTag = true
	default:
		tagArgs = constructTagArgs(userArgs)
	}
	finalArgs := append(tagArgs, userArgs...)

	cmd := exec.Command("rg", finalArgs...)

	if disableTag || !isatty(os.Stdin) || !isatty(os.Stdout) {
		// Data being piped from stdin
		os.Exit(passThrough(cmd))
	}

	handleColorSetting(finalArgs)
	os.Exit(generateTags(cmd))
}
