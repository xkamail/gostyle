package main

import (
	"embed"
	"flag"
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
)

//go:embed skills
var skillsFS embed.FS

// runSkills writes the embedded skills under -dir. It overwrites the files it ships and
// leaves everything else alone, so a repo's references/local.md survives a reinstall.
func runSkills(args []string) {
	if len(args) == 0 || args[0] != "install" {
		fmt.Fprintln(os.Stderr, "usage: gostyle skills install [-dir .claude/skills]")
		os.Exit(2)
	}
	flags := flag.NewFlagSet("gostyle skills install", flag.ExitOnError)
	dir := flags.String("dir", ".claude/skills", "directory to install the skills into")
	_ = flags.Parse(args[1:])
	err := fs.WalkDir(skillsFS, "skills", func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		rel, err := filepath.Rel("skills", p)
		if err != nil {
			return err
		}
		dst := filepath.Join(*dir, rel)
		data, err := skillsFS.ReadFile(p)
		if err != nil {
			return err
		}
		data = []byte(strings.Replace(string(data), "<!-- Installed by `go tool gostyle skills install`",
			"<!-- Installed by gostyle "+version()+" (`go tool gostyle skills install`)", 1))
		if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
			return err
		}
		if err := os.WriteFile(dst, data, 0o644); err != nil {
			return err
		}
		fmt.Println(dst)
		return nil
	})
	if err != nil {
		slog.Error("install skills", "error", err)
		os.Exit(1)
	}
}
