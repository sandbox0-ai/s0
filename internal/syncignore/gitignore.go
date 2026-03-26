package syncignore

import (
	"bufio"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

// Matcher evaluates root-level .gitignore rules for local upload filtering.
type Matcher struct {
	rules []rule
}

type rule struct {
	baseDir  string
	negated  bool
	dirOnly  bool
	anchored bool
	pattern  string
	matcher  *regexp.Regexp
	basename bool
}

// Load reads root and nested .gitignore rules plus .git/info/exclude when present.
func Load(workspaceRoot string) (*Matcher, error) {
	matcher := &Matcher{}
	root := strings.ReplaceAll(workspaceRoot, "\\", "/")
	if err := os.MkdirAll(workspaceRoot, 0o755); err != nil {
		return nil, err
	}

	for _, relative := range []string{".gitignore", ".git/info/exclude"} {
		if err := loadRuleFile(matcher, root, relative); err != nil {
			return nil, err
		}
	}

	var ignoreFiles []string
	err := filepath.WalkDir(workspaceRoot, func(current string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() && entry.Name() == ".git" {
			return filepath.SkipDir
		}
		if entry.IsDir() || entry.Name() != ".gitignore" || current == filepath.Join(workspaceRoot, ".gitignore") {
			return nil
		}
		relative, err := filepath.Rel(workspaceRoot, current)
		if err != nil {
			return err
		}
		ignoreFiles = append(ignoreFiles, filepath.ToSlash(relative))
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Strings(ignoreFiles)
	for _, relative := range ignoreFiles {
		if err := loadRuleFile(matcher, root, relative); err != nil {
			return nil, err
		}
	}
	return matcher, nil
}

// Match reports whether a relative workspace path should be ignored for local uploads.
func (m *Matcher) Match(relativePath string, isDir bool) bool {
	relativePath = normalizePath(relativePath)
	if relativePath == "" || relativePath == "." {
		return false
	}

	ignored := false
	for _, rule := range m.rules {
		if rule.matches(relativePath) {
			ignored = !rule.negated
		}
	}
	return ignored
}

func parseRule(line string) (rule, bool) {
	parsed := rule{}
	if strings.HasPrefix(line, "!") {
		parsed.negated = true
		line = strings.TrimSpace(strings.TrimPrefix(line, "!"))
	}
	if line == "" {
		return rule{}, false
	}

	if strings.HasPrefix(line, "/") {
		parsed.anchored = true
		line = strings.TrimPrefix(line, "/")
	}
	if strings.HasSuffix(line, "/") {
		parsed.dirOnly = true
		line = strings.TrimSuffix(line, "/")
	}
	line = normalizePath(line)
	if line == "" {
		return rule{}, false
	}
	parsed.pattern = line
	parsed.basename = !strings.Contains(line, "/")

	re, err := regexp.Compile(buildPatternRegex(line, parsed.basename, parsed.dirOnly, parsed.anchored))
	if err != nil {
		return rule{}, false
	}
	parsed.matcher = re
	return parsed, true
}

func (r rule) matches(relativePath string) bool {
	if r.matcher == nil {
		return false
	}
	if r.baseDir != "" {
		if relativePath != r.baseDir && !strings.HasPrefix(relativePath, r.baseDir+"/") {
			return false
		}
		relativePath = strings.TrimPrefix(strings.TrimPrefix(relativePath, r.baseDir), "/")
	}
	return r.matcher.MatchString(relativePath)
}

func buildPatternRegex(pattern string, basename, dirOnly, anchored bool) string {
	var prefix string
	switch {
	case anchored:
		prefix = `^`
	case basename:
		prefix = `(^|.*/)`
	default:
		prefix = `(^|.*/)`
	}
	var suffix string
	if dirOnly {
		suffix = `(/.*)?$`
	} else {
		suffix = `$`
	}
	return prefix + globToRegex(pattern) + suffix
}

func globToRegex(pattern string) string {
	var builder strings.Builder
	for i := 0; i < len(pattern); i++ {
		switch pattern[i] {
		case '*':
			if i+1 < len(pattern) && pattern[i+1] == '*' {
				builder.WriteString(".*")
				i++
				continue
			}
			builder.WriteString(`[^/]*`)
		case '?':
			builder.WriteString(`[^/]`)
		case '.', '+', '(', ')', '|', '^', '$', '{', '}', '[', ']', '\\':
			builder.WriteByte('\\')
			builder.WriteByte(pattern[i])
		default:
			builder.WriteByte(pattern[i])
		}
	}
	return builder.String()
}

func normalizePath(value string) string {
	value = strings.ReplaceAll(value, "\\", "/")
	value = path.Clean(value)
	if value == "." {
		return ""
	}
	return strings.TrimPrefix(value, "./")
}

func loadRuleFile(matcher *Matcher, workspaceRoot, relativePath string) error {
	file, err := os.Open(path.Join(workspaceRoot, relativePath))
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	defer file.Close()

	baseDir := normalizePath(path.Dir(relativePath))
	if relativePath == ".git/info/exclude" {
		baseDir = ""
	}
	if baseDir == "." {
		baseDir = ""
	}
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if parsed, ok := parseRule(line); ok {
			parsed.baseDir = baseDir
			matcher.rules = append(matcher.rules, parsed)
		}
	}
	return scanner.Err()
}
