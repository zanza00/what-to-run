// The justfile source: just recipes.
package core

import (
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"slices"
	"strconv"
	"strings"
)

// justfileSource returns the source for just recipes. It reads the first of the
// three file names that just itself looks for, in that order. Like just, it also
// looks in the parent directories of the project directory and follows the
// local imports and modules of the file.
func justfileSource(files Files) source {
	return source{
		SourceInfo: SourceInfo{
			ID:            "just",
			Label:         "justfile",
			Files:         []string{"justfile", "Justfile", ".justfile"},
			SearchParents: true,
		},
		Parse: func(dir string, data []byte) ([]Task, error) {
			return parseJustfileFor(runtime.GOOS, files, dir, data)
		},
	}
}

// justRecipeRe matches a recipe header at the start of a line: an optional "@"
// that marks a silent recipe, the recipe name, then anything that is not a
// colon (parameters and dependencies) up to the colon that opens the recipe.
var justRecipeRe = regexp.MustCompile(`^@?([A-Za-z_][A-Za-z0-9_-]*)[^:]*:`)

// justKeywordRe matches the set, alias and export keywords. just takes any
// whitespace after a keyword, a TAB included, and then either a statement or a
// recipe header that carries the keyword as its name ("set:"). justHeaderColon
// tells the two apart.
var justKeywordRe = regexp.MustCompile(`^(set|alias|export)\s`)

// justModuleRe matches the import and mod keywords, including their optional
// "?" forms, which take a path up to whitespace. The whitespace is required,
// for the same reason as in justKeywordRe.
var justModuleRe = regexp.MustCompile(`^(import|mod)\??\s`)

// justImportRe captures the rest of an import statement, which
// justImportStatement reads as a quoted path and an optional comment.
var justImportRe = regexp.MustCompile(`^import\??\s+(.*)$`)

// justModRe captures the rest of a module statement, which justModuleStatement
// reads as a module name and an optional explicit path.
var justModRe = regexp.MustCompile(`^mod\??\s+(.*)$`)

// justModuleNameRe splits the rest of a module statement into the module name
// and the unparsed remainder. The remainder must be empty or a quoted path.
var justModuleNameRe = regexp.MustCompile(`^([A-Za-z_][A-Za-z0-9_-]*)(?:\s+(.*))?$`)

// justAssignmentRe matches a top-level variable assignment. It is anchored and
// requires the ":=" right after the name, so the parser does not mistake a
// recipe header with a trailing comment ("build: # see FOO:=1") or a parameter
// default that contains ":=" for an assignment.
var justAssignmentRe = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_-]*\s*:=`)

// parseJustfileFor returns the recipes that a justfile declares for goos,
// sorted by name like the ungrouped part of "just --list". Each recipe runs as
// "just <name>". Imported recipes are listed as if the root file declared them.
//
// The parser is line-based and forgiving. It recognizes a recipe header at
// column 0. It keeps the last contiguous "#" comment line above the header, or
// the "[doc(...)]" attribute of the recipe, as the description. It hides a
// private recipe: the "_" prefix and the "[private]" attribute.
//
// Platform attributes filter a recipe. [windows], [linux], [macos] and [unix]
// list the recipe only on a matching GOOS, where unix means any GOOS but
// Windows, and several platform attributes are OR-ed. Other attributes do not
// affect the list.
//
// A name may have several definitions, and only one row appears. just sees the
// first definition that is enabled on goos, so its doc and its privacy decide
// the row. A name with no enabled definition is dropped. So a recipe defined
// under [unix] and again under [windows] shows one row, with the doc of the
// active definition.
//
// A doc attribute always decides, even when its text is empty or blank. just
// shows no description then, so the "#" comment above does not apply. An
// attribute list may carry a trailing "#" comment and may list several
// attributes separated by commas, as in [private, group("g")] # why. The
// scanner ignores commas and "#" inside a quoted doc string, reads "private"
// and "doc(...)", and leaves any other attribute alone.
//
// The list may also span several lines, as just allows ([private, on one line
// and group("g")] on the next, indented). The parser buffers its lines, blank
// ones included, until the "]" outside a quoted string, so it does not mistake
// an indented continuation for a recipe body. A leading "#!" is a shebang, not
// a doc comment. The parser ignores the lines inside recipe bodies.
//
// A recipe header may continue on the next line. A column-0 line that ends with
// a backslash, after any trailing spaces or tabs, joins the following line,
// indented or not, with a single space. Several continuations join in turn, so
// "foo \" and "  bar:" declare the recipe "foo bar".
//
// An indented line never continues. A shell line that ends with a backslash
// inside a recipe body stays a body line and does not swallow the recipe below
// it. A "#" comment is never continued either, whether it opens the line or
// follows a header ("build: # note \"). just keeps that backslash as comment
// text, so the recipe below it stays a recipe. The same rule applies to a
// backslash left inside an unterminated quoted string. The parser skips a
// continued setting or assignment line like the one-line form, and it drops a
// continuation that the file never closes.
//
// The parser follows imports and modules recursively. "import" and "import?"
// take a quoted path, resolved against the directory of the declaring file,
// absolute paths and a leading "~/" included. The imported recipes keep the
// namespace of the importing file, as they do in just. "mod" and "mod?" name a
// module whose recipes are listed as "name::recipe", the form that just accepts
// on the command line. The source of the module is name.just or, inside the
// name/ directory, mod.just or justfile/.justfile in any capitalization, and an
// explicit quoted path may name the file or its directory.
//
// The parser skips a file that is missing, unreadable or unparsable. It does
// not read a file that is already on the active chain, so a cycle terminates
// and the picker never fails on an include. A nil Files port stops at the root
// document, which is what the parser tests use.
//
// When more than one file defines a name, the shallowest definition wins and,
// at equal depth, the first one met in source order. That is the rule that just
// documents for duplicate recipes. The platform and privacy attributes still
// filter the definitions before that choice, so only the definitions enabled on
// goos take part in it. A module is a namespace of its own, so its recipes do
// not collide with root or imported ones.
//
// The parser skips settings, aliases and exports, as it does the top-level
// variable assignments. The assignment rule is anchored on the name and the
// ":=", so a recipe header with a trailing comment or a parameter default that
// contains ":=" is still listed. It skips a keyword statement only while no
// recipe-header colon follows, so a recipe named "set", "alias", "export",
// "import" or "mod" is still listed, however its parameters look. Aliases are
// not listed as extra rows. Groups are ignored: the picker shows one flat
// alphabetical list, which orders grouped recipes differently from "just
// --list". Triple-quoted strings can contain a column-0 line that looks like a
// recipe header, and the line-based parser cannot tell it from a real one.
func parseJustfileFor(goos string, files Files, dir string, data []byte) ([]Task, error) {
	return selectJustTasks(goos, justCollect(files, dir, "", data, 0, nil)), nil
}

// justDocument is one parsed justfile: the recipes it defines, the local
// justfiles it imports and the modules it declares, all in source order.
type justDocument struct {
	defs    []justRecipe
	imports []justImport
	mods    []justModule
}

// justImport is one "import" statement: the path as written, still unresolved.
// The optional "import?" form needs no flag here, because the parser skips a
// missing import either way.
type justImport struct {
	path string
}

// justModule is one "mod" statement: the module name and, when given, the
// explicit source path as written. The optional "mod?" form needs no flag here,
// because the parser skips a missing module either way.
type justModule struct {
	name string
	path string
}

// justDocumentOf parses one justfile document into the recipes it defines, the
// local files it imports and the modules it declares, in source order. It keeps
// every duplicate and platform variant.
func justDocumentOf(data []byte) justDocument {
	var (
		defs       []justRecipe
		imports    []justImport
		mods       []justModule
		comment    string   // last contiguous "#" line
		docAttr    string   // "[doc(...)]" attribute
		hasDocAttr bool     // whether a "[doc(...)]" attribute was seen
		private    bool     // "[private]" attribute
		platforms  []string // platform attributes, OR-ed
		attrParts  []string // lines of an attribute list not closed yet
	)

	reset := func() {
		comment, docAttr, hasDocAttr, private, platforms, attrParts = "", "", false, false, nil, nil
	}

	first := true
	lines := slices.Collect(strings.Lines(string(data)))
	for i := 0; i < len(lines); i++ {
		line, next, ok := justLogicalLine(lines, i)
		if !ok {
			// The file never closes this continuation, so the parser cannot
			// read it. Ignore it, with the attributes buffered above it.
			reset()
			break
		}
		i = next - 1

		// A shebang describes the interpreter, not the recipe below it.
		if first {
			first = false
			if strings.HasPrefix(line, "#!") {
				continue
			}
		}
		if strings.TrimSpace(line) == "" {
			// A blank line inside an attribute list belongs to the list. just
			// keeps the list, and the comment block above it, across it.
			if len(attrParts) > 0 {
				continue
			}
			reset()
			continue
		}
		if rest, ok := strings.CutPrefix(line, "#"); ok {
			comment = strings.TrimSpace(rest)
			continue
		}
		// An attribute list opens at column 0 and may span several lines. just
		// closes it at the "]" outside a quoted string, and the continuation
		// lines may be indented, so the parser must not read them as a recipe
		// body. A list that the file never closes is dropped.
		if line[0] == '[' || len(attrParts) > 0 {
			attrParts = append(attrParts, stripJustComment(line))
			attrs := strings.Join(attrParts, " ")
			end := justAttributeEnd(attrs)
			if end < 0 {
				continue
			}
			attrParts = nil
			parsed := justAttributes(attrs[:end+1])
			if parsed.hasDoc {
				// The flag is set even for an empty text. just then shows no
				// description, rather than the comment above it.
				docAttr, hasDocAttr = parsed.doc, true
			}
			private = private || parsed.private
			platforms = append(platforms, parsed.platforms...)
			// Other attributes, like [group("...")], leave the pending
			// description and privacy alone.
			continue
		}
		if line[0] == ' ' || line[0] == '\t' {
			continue // a line inside a recipe body
		}
		// The parser checks settings, assignments, aliases, exports, imports
		// and modules before the recipe regex. An import path or an assignment
		// that contains a colon must not become a recipe header.
		if isJustSetting(line) {
			if path, ok := justImportStatement(line); ok {
				imports = append(imports, justImport{path: path})
			} else if mod, ok := justModuleStatement(line); ok {
				mods = append(mods, mod)
			}
			reset()
			continue
		}

		m := justRecipeRe.FindStringSubmatch(line)
		if m == nil {
			reset()
			continue
		}
		name := m[1]
		// A doc attribute decides the description whenever one was seen, even
		// with an empty or blank text. just shows no description there, so the
		// "#" comment above it does not come back. The comment applies only
		// without a doc attribute.
		var desc string
		if hasDocAttr {
			if strings.TrimSpace(docAttr) != "" {
				desc = docAttr
			}
		} else {
			desc = comment
		}
		// A recipe may be defined once per platform, so keep every definition
		// until the file is read. Only then can the parser pick the first
		// variant that is enabled on goos.
		defs = append(defs, justRecipe{
			name: name,
			desc: desc,
			// The underscore prefix hides a recipe just like [private] does, and
			// the parser decides that here. A module prefixes the name later,
			// so the selection cannot look at the start of the final name.
			private:   private || strings.HasPrefix(name, "_"),
			platforms: platforms,
		})
		// just clears the pending attributes after every recipe header, even
		// with no blank line between two recipes. A [doc] or [private] recipe
		// must not leak its description or privacy onto the next one.
		reset()
	}

	return justDocument{defs: defs, imports: imports, mods: mods}
}

// justCollect gathers the definitions of one justfile, of the files it imports
// and of its modules, recursively. prefix is the namespace inherited from the
// enclosing module ("name::"), empty for the root file. depth is the number of
// include hops from the root file. active holds the absolute paths on the
// current include chain. A cycle is skipped, while the same file reached from
// two branches is still read under both.
func justCollect(files Files, dir, prefix string, data []byte, depth int, active map[string]bool) []justDefinition {
	doc := justDocumentOf(data)
	defs := make([]justDefinition, 0, len(doc.defs))
	for _, def := range doc.defs {
		def.name = prefix + def.name
		defs = append(defs, justDefinition{justRecipe: def, depth: depth})
	}
	// Without a Files port there is nothing to read, so the parser tests stop
	// at the root document.
	if files == nil || (len(doc.imports) == 0 && len(doc.mods) == 0) {
		return defs
	}
	if active == nil {
		active = make(map[string]bool)
	}
	for _, imp := range doc.imports {
		path := justImportPath(dir, imp.path)
		abs, err := filepath.Abs(path)
		if err != nil || active[abs] {
			continue
		}
		imported, err := files.ReadFile(path)
		if err != nil {
			continue
		}
		active[abs] = true
		defs = append(defs, justCollect(files, filepath.Dir(path), prefix, imported, depth+1, active)...)
		delete(active, abs)
	}
	for _, mod := range doc.mods {
		path, imported, ok := justModuleFile(files, dir, mod)
		if !ok {
			continue
		}
		abs, err := filepath.Abs(path)
		if err != nil || active[abs] {
			continue
		}
		active[abs] = true
		defs = append(defs, justCollect(files, filepath.Dir(path), prefix+mod.name+"::", imported, depth+1, active)...)
		delete(active, abs)
	}
	return defs
}

// justImportPath resolves an import path the way just does. The path is
// relative to the directory of the importing file, unless it is absolute, and a
// leading "~/" names the home directory of the user.
func justImportPath(dir, path string) string {
	if rest, ok := strings.CutPrefix(path, "~/"); ok {
		if home, err := os.UserHomeDir(); err == nil {
			return filepath.Join(home, rest)
		}
	}
	if filepath.IsAbs(path) {
		return path
	}
	return filepath.Join(dir, path)
}

// justModuleFile reads the source of one module. An explicit path is read as a
// file, or as a directory that holds mod.just, justfile or .justfile. Without a
// path, the candidates of just are name.just, then name/mod.just, name/justfile
// and name/.justfile, the last two in any capitalization.
func justModuleFile(files Files, dir string, mod justModule) (string, []byte, bool) {
	if mod.path != "" {
		base := justImportPath(dir, mod.path)
		if data, err := files.ReadFile(base); err == nil {
			return base, data, true
		}
		return justModuleDirFile(files, base)
	}
	file := filepath.Join(dir, mod.name+".just")
	if data, err := files.ReadFile(file); err == nil {
		return file, data, true
	}
	return justModuleDirFile(files, filepath.Join(dir, mod.name))
}

// justModuleDirFile finds a module source inside dir: mod.just first, then
// justfile or .justfile, the names that just accepts.
func justModuleDirFile(files Files, dir string) (string, []byte, bool) {
	entries, err := files.ReadDir(dir)
	if err != nil {
		return "", nil, false
	}
	for _, want := range []string{"mod.just", "justfile", ".justfile"} {
		for _, entry := range entries {
			if entry.IsDir() || !justModuleName(entry.Name(), want) {
				continue
			}
			path := filepath.Join(dir, entry.Name())
			if data, err := files.ReadFile(path); err == nil {
				return path, data, true
			}
		}
	}
	return "", nil, false
}

// justModuleName reports whether a directory entry carries the wanted module
// file name. mod.just is exact, while justfile and .justfile match in any
// capitalization, as just allows.
func justModuleName(name, want string) bool {
	if name == want {
		return true
	}
	return want != "mod.just" && strings.EqualFold(name, want)
}

// justModuleStatement returns the module of a "mod" or "mod?" statement. The
// rest is a module name and, optionally, a quoted explicit path. Anything else
// declares no module. The caller has already told the line apart from a recipe
// named "mod".
func justModuleStatement(line string) (justModule, bool) {
	m := justModRe.FindStringSubmatch(line)
	if m == nil {
		return justModule{}, false
	}
	parts := justModuleNameRe.FindStringSubmatch(strings.TrimSpace(stripJustComment(m[1])))
	if parts == nil {
		return justModule{}, false
	}
	mod := justModule{name: parts[1]}
	rest := strings.TrimSpace(parts[2])
	if rest == "" {
		return mod, true
	}
	path, ok := unquoteJustString(rest)
	if !ok {
		return justModule{}, false
	}
	mod.path = path
	return mod, true
}

// selectJustTasks picks one definition per recipe name and turns it into a
// task. The shallowest definition enabled on goos decides, and at equal depth
// the first one met in source order wins. The privacy of that definition hides
// the row, and the function drops definitions of other platforms and later
// duplicates.
func selectJustTasks(goos string, defs []justDefinition) []Task {
	best := make(map[string]justDefinition, len(defs))
	for _, def := range defs {
		if !justPlatformEnabled(def.platforms, goos) {
			continue
		}
		if prev, ok := best[def.name]; ok && prev.depth <= def.depth {
			continue
		}
		best[def.name] = def
	}

	tasks := make([]Task, 0, len(best))
	for _, def := range best {
		// Hiding applies to the selected definition only, as in just: the
		// winning variant is the one the tool would see.
		if def.private {
			continue
		}
		tasks = append(tasks, Task{
			Name: def.name,
			Desc: def.desc,
			Args: []string{"just", def.name},
		})
	}

	if len(tasks) == 0 {
		return nil
	}
	slices.SortFunc(tasks, func(a, b Task) int { return strings.Compare(a.Name, b.Name) })
	return tasks
}

// justRecipe is one definition of a recipe in a justfile. The same name may
// appear more than once with different platform attributes, in one file or in
// imported ones.
type justRecipe struct {
	name      string
	desc      string
	private   bool     // [private] or the "_" prefix
	platforms []string // empty means every platform
}

// justDefinition is one recipe definition with the import depth of the file
// that declares it: the root justfile is 0.
type justDefinition struct {
	justRecipe
	depth int
}

// justPlatformEnabled reports whether a recipe with these platform attributes
// is visible on goos. No attribute means every platform. Several attributes are
// OR-ed, as in just, and "unix" covers every GOOS but Windows.
func justPlatformEnabled(attrs []string, goos string) bool {
	if len(attrs) == 0 {
		return true
	}
	for _, attr := range attrs {
		switch attr {
		case "windows":
			if goos == "windows" {
				return true
			}
		case "linux":
			if goos == "linux" {
				return true
			}
		case "macos":
			if goos == "darwin" {
				return true
			}
		case "unix":
			if goos != "windows" {
				return true
			}
		}
	}
	return false
}

// justLogicalLine returns the logical line that starts at lines[i] and the
// index of the first line after it. A line that starts at column 0 and ends
// with a backslash, after any trailing spaces or tabs, continues on the next
// line, which may be indented. just joins the two with a space, and several
// continuations join in turn.
//
// Only a column-0 line continues. An indented line is a recipe body line, and a
// shell line that ends with a backslash there must stay one instead of
// swallowing the recipe below it. A "#" comment does not continue either,
// because just keeps the backslash in the comment text, whether the "#" opens
// the line or follows a header, as in "build: # note \". A backslash left
// inside an unterminated quoted string is also text rather than a continuation.
// ok is false when the continuation runs off the end of the file.
func justLogicalLine(lines []string, i int) (line string, next int, ok bool) {
	line = strings.TrimRight(lines[i], "\r\n")
	next = i + 1
	if line == "" || line[0] == ' ' || line[0] == '\t' || line[0] == '#' {
		return line, next, true
	}
	for {
		text := strings.TrimRight(line, " \t")
		if !justContinues(text) {
			return line, next, true
		}
		if next >= len(lines) {
			return "", next, false
		}
		left := strings.TrimRight(text[:len(text)-1], " \t")
		cont := strings.TrimSpace(strings.TrimRight(lines[next], "\r\n"))
		next++
		// A line carrying nothing but the backslash leaves the continuation
		// at column 0, as just reads it.
		if left == "" {
			line = cont
		} else {
			line = left + " " + cont
		}
		// The joined line may itself continue. A "#" or an empty continuation
		// ends the join.
		if line == "" || line[0] == '#' {
			return line, next, true
		}
	}
}

// justContinues reports whether the right-trimmed line ends with a backslash
// that continues it on the next line. The backslash counts only outside a
// quoted string and outside a "#" comment. just keeps the trailing backslash of
// "build: # note \" as comment text, and the scanner cannot tell an unterminated
// string from a continuation either, so it leaves both alone. Quoted strings
// and escapes are read like justHeaderColon reads them.
func justContinues(text string) bool {
	if !strings.HasSuffix(text, `\`) {
		return false
	}
	var quote byte
	for i := 0; i < len(text)-1; i++ {
		c := text[i]
		switch {
		case quote != 0:
			if c == '\\' {
				i++ // skip the escaped byte
				continue
			}
			if c == quote {
				quote = 0
			}
		case c == '"' || c == '\'':
			quote = c
		case c == '#':
			return false
		}
	}
	return quote == 0
}

// isJustSetting reports whether the line declares a setting, an assignment, an
// alias, an export, an import or a module rather than a recipe. A keyword
// statement must be followed by whitespace (just reads a TAB as well as a
// space) and must carry no colon that opens a recipe header. just lets a recipe
// take a keyword as its name ("set x="1":"), and the header colon tells that
// recipe apart from a setting.
func isJustSetting(line string) bool {
	if justAssignmentRe.MatchString(line) {
		return true
	}
	if justKeywordRe.MatchString(line) || justModuleRe.MatchString(line) {
		return !justHeaderColon(line)
	}
	return false
}

// justImportStatement returns the path of an "import" or "import?" statement.
// The path is a quoted string, optionally followed by a "#" comment. Anything
// else after the keyword declares no import. The caller has already told the
// line apart from a recipe named "import".
func justImportStatement(line string) (string, bool) {
	m := justImportRe.FindStringSubmatch(line)
	if m == nil {
		return "", false
	}
	return unquoteJustString(strings.TrimSpace(stripJustComment(m[1])))
}

// justHeaderColon reports whether the line carries the colon that opens a
// recipe header, which is a ":" outside a quoted string and not part of a ":="
// assignment. A "#" starts a comment that may contain any text, so the scan
// stops there. Quoted strings are read like stripJustComment reads them, so the
// ":" of an import path or of a parameter default does not count.
func justHeaderColon(line string) bool {
	var quote byte
	for i := 0; i < len(line); i++ {
		c := line[i]
		switch {
		case quote != 0:
			if c == '\\' {
				i++ // skip the escaped byte
				continue
			}
			if c == quote {
				quote = 0
			}
		case c == '"' || c == '\'':
			quote = c
		case c == '#':
			return false
		case c == ':':
			if i+1 < len(line) && line[i+1] == '=' {
				i++ // skip the "=" of ":="
				continue
			}
			return true
		}
	}
	return false
}

// justAttrs is the part of a "[...]" attribute list that the picker uses: the
// doc text, whether a doc attribute was present, privacy, and the platform
// attributes that filter a recipe.
type justAttrs struct {
	doc       string
	hasDoc    bool
	private   bool
	platforms []string
}

// justAttributes parses a "[...]" attribute list and returns its doc text, if
// the list carries a "doc(...)", whether it marks the recipe private, and its
// platform attributes. Only [windows], [linux], [macos] and [unix] are platform
// attributes, and the parser ignores the other attributes. The text may be a
// single line or the lines of a multi-line list that justDocumentOf joined with
// spaces. It may carry a trailing "#" comment and several comma-separated
// attributes, as in [private, group("g")] # why. Commas and "#" inside a quoted
// doc string are not separators.
func justAttributes(list string) justAttrs {
	body, ok := strings.CutPrefix(strings.TrimSpace(stripJustComment(list)), "[")
	if !ok {
		return justAttrs{}
	}
	body, ok = strings.CutSuffix(strings.TrimSpace(body), "]")
	if !ok {
		return justAttrs{}
	}

	var parsed justAttrs
	for _, attr := range splitJustAttributes(body) {
		attr = strings.TrimSpace(attr)
		switch {
		case attr == "private":
			parsed.private = true
		case attr == "windows" || attr == "linux" || attr == "macos" || attr == "unix":
			parsed.platforms = append(parsed.platforms, attr)
		case strings.HasPrefix(attr, "doc("):
			if text, ok := justDocText(attr); ok {
				parsed.doc, parsed.hasDoc = text, true
			}
		}
	}
	return parsed
}

// justAttributeEnd returns the index of the "]" that closes an attribute list,
// or -1 while the list is still open. A "]" inside a quoted string does not
// close it, so the parser reads a bracket in a doc string like any other
// character.
func justAttributeEnd(text string) int {
	var quote byte
	for i := 0; i < len(text); i++ {
		c := text[i]
		switch {
		case quote != 0:
			if c == '\\' {
				i++ // skip the escaped byte
				continue
			}
			if c == quote {
				quote = 0
			}
		case c == '"' || c == '\'':
			quote = c
		case c == ']':
			return i
		}
	}
	return -1
}

// stripJustComment drops a trailing "#" comment from an attribute line and
// ignores a "#" inside a quoted string.
func stripJustComment(line string) string {
	var quote byte
	for i := 0; i < len(line); i++ {
		c := line[i]
		switch {
		case quote != 0:
			if c == '\\' {
				i++ // skip the escaped byte
				continue
			}
			if c == quote {
				quote = 0
			}
		case c == '"' || c == '\'':
			quote = c
		case c == '#':
			return line[:i]
		}
	}
	return line
}

// splitJustAttributes splits an attribute list on the commas that sit outside
// a quoted string, so the commas of a doc string do not split it.
func splitJustAttributes(attrs string) []string {
	var (
		parts []string
		start int
		quote byte
	)
	for i := 0; i < len(attrs); i++ {
		c := attrs[i]
		switch {
		case quote != 0:
			if c == '\\' {
				i++ // skip the escaped byte
				continue
			}
			if c == quote {
				quote = 0
			}
		case c == '"' || c == '\'':
			quote = c
		case c == ',':
			parts = append(parts, attrs[start:i])
			start = i + 1
		}
	}
	return append(parts, attrs[start:])
}

// justDocText returns the text of one "doc(...)" attribute and unquotes the
// string. It unescapes escaped quotes, as just reads them.
func justDocText(attr string) (string, bool) {
	rest, ok := strings.CutPrefix(attr, "doc(")
	if !ok {
		return "", false
	}
	rest, ok = strings.CutSuffix(rest, ")")
	if !ok {
		return "", false
	}
	return unquoteJustString(strings.TrimSpace(rest))
}

// unquoteJustString unquotes a just string literal. Double-quoted strings use
// backslash escapes, and single-quoted strings are raw except for the escaped
// quote.
func unquoteJustString(s string) (string, bool) {
	if len(s) < 2 || s[0] != s[len(s)-1] || (s[0] != '"' && s[0] != '\'') {
		return "", false
	}
	if s[0] == '"' {
		if text, err := strconv.Unquote(s); err == nil {
			return text, true
		}
		return s[1 : len(s)-1], true
	}
	return strings.ReplaceAll(s[1:len(s)-1], `\'`, `'`), true
}
