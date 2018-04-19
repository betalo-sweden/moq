package moq

import (
	"bytes"
	"errors"
	"fmt"
	"go/ast"
	"go/build"
	"go/parser"
	"go/token"
	"go/types"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"
	"text/template"

	"golang.org/x/tools/go/loader"
	"golang.org/x/tools/imports"
)

// This list comes from the golint codebase. Golint will complain about any of
// these being mixed-case, like "Id" instead of "ID".
var golintInitialisms = []string{
	"ACL",
	"API",
	"ASCII",
	"CPU",
	"CSS",
	"DNS",
	"EOF",
	"GUID",
	"HTML",
	"HTTP",
	"HTTPS",
	"ID",
	"IP",
	"JSON",
	"LHS",
	"QPS",
	"RAM",
	"RHS",
	"RPC",
	"SLA",
	"SMTP",
	"SQL",
	"SSH",
	"TCP",
	"TLS",
	"TTL",
	"UDP",
	"UI",
	"UID",
	"UUID",
	"URI",
	"URL",
	"UTF8",
	"VM",
	"XML",
	"XMPP",
	"XSRF",
	"XSS",
}

// Mocker can generate mock structs.
type Mocker struct {
	src     string
	tmpl    *template.Template
	fset    *token.FileSet
	pkgs    map[string]*ast.Package
	pkgName string
	pkgPath string

	imports map[string]bool
}

// New makes a new Mocker for the specified package directory.
func New(src, packageName string) (*Mocker, error) {
	fset := token.NewFileSet()
	noTestFiles := func(i os.FileInfo) bool {
		return !strings.HasSuffix(i.Name(), "_test.go")
	}

	pkgs, err := parser.ParseDir(fset, src, noTestFiles, parser.SpuriousErrors)
	if err != nil {
		return nil, err
	}
	if len(packageName) == 0 {

		for pkgName := range pkgs {
			if strings.Contains(pkgName, "_test") {
				continue
			}
			packageName = pkgName
			break
		}
	}
	if len(packageName) == 0 {
		return nil, errors.New("failed to determine package name")
	}
	pkgPath, err := pkgPath(src)
	if err != nil {
		return nil, err
	}
	pkgPath = filepath.Join(filepath.Dir(pkgPath), packageName)

	tmpl, err := template.New("moq").Funcs(templateFuncs).Parse(moqTemplate)
	if err != nil {
		return nil, err
	}
	return &Mocker{
		src:     src,
		tmpl:    tmpl,
		fset:    fset,
		pkgs:    pkgs,
		pkgName: packageName,
		pkgPath: pkgPath,
		imports: make(map[string]bool),
	}, nil
}

func removeOldMocks(srcFolder string) error {
	abs, err := filepath.Abs(srcFolder)
	if err != nil {
		return err
	}
	oldMocks, err := filepath.Glob(filepath.Join(abs, "*_mock.go"))
	if err != nil {
		return err
	}
	for _, mock := range oldMocks {
		os.Remove(mock)
	}
	return nil
}

// Mock generates a mock for the specified interface name.
func (m *Mocker) Mock(w io.Writer, name ...string) error {
	if len(name) == 0 {
		return errors.New("must specify one interface")
	}

	if err := removeOldMocks(m.src); err != nil {
		return errors.New("failed to clean up old mocks")
	}

	pkgInfo, err := m.pkgInfoFromPath(m.src)
	if err != nil {
		return err
	}

	doc := doc{
		PackageName: m.pkgName,
		Imports:     moqImports,
	}

	mocksMethods := false

	tpkg := pkgInfo.Pkg
	for _, n := range name {
		iface := tpkg.Scope().Lookup(n)
		if iface == nil {
			return fmt.Errorf("cannot find interface %s", n)
		}
		if !types.IsInterface(iface.Type()) {
			return fmt.Errorf("%s (%s) not an interface", n, iface.Type().String())
		}
		iiface := iface.Type().Underlying().(*types.Interface).Complete()
		obj := obj{
			InterfaceName: n,
		}
		for i := 0; i < iiface.NumMethods(); i++ {
			mocksMethods = true
			meth := iiface.Method(i)
			sig := meth.Type().(*types.Signature)
			method := &method{
				Name: meth.Name(),
			}
			obj.Methods = append(obj.Methods, method)
			method.Params = m.extractArgs(sig, sig.Params(), "in%d")
			method.Returns = m.extractArgs(sig, sig.Results(), "out%d")
		}
		doc.Objects = append(doc.Objects, obj)
	}

	if mocksMethods {
		doc.Imports = append(doc.Imports, "sync")
	}

	for pkgToImport := range m.imports {
		doc.Imports = append(doc.Imports, stripVendorPath(pkgToImport))
	}

	var buf bytes.Buffer
	err = m.tmpl.Execute(&buf, doc)
	if err != nil {
		return err
	}
	formatted, err := imports.Process("", buf.Bytes(), nil)
	if err != nil {
		return fmt.Errorf("goimports: %s", err)
	}
	if _, err := w.Write(formatted); err != nil {
		return err
	}
	return nil
}

func (m *Mocker) packageQualifier(pkg *types.Package) string {
	path := pkg.Path()
	importPath := filepath.Join(filepath.Dir(path), pkg.Name())

	if importPath == m.pkgPath {
		return ""
	}
	if path == "." {
		path = m.pkgPath
	}
	m.imports[path] = true
	return pkg.Name()
}

func (m *Mocker) extractArgs(sig *types.Signature, list *types.Tuple, nameFormat string) []*param {
	var params []*param
	listLen := list.Len()
	for ii := 0; ii < listLen; ii++ {
		p := list.At(ii)
		name := p.Name()
		if name == "" {
			name = fmt.Sprintf(nameFormat, ii+1)
		}
		typename := types.TypeString(p.Type(), m.packageQualifier)
		// check for final variadic argument
		variadic := sig.Variadic() && ii == listLen-1 && typename[0:2] == "[]"
		param := &param{
			Name:     name,
			Type:     typename,
			Variadic: variadic,
		}
		params = append(params, param)
	}
	return params
}

func (*Mocker) pkgInfoFromPath(src string) (*loader.PackageInfo, error) {
	pkgFull, err := pkgPath(src)
	if err != nil {
		return nil, err
	}

	conf := loader.Config{
		ParserMode: parser.SpuriousErrors,
		Cwd:        src,
	}
	conf.Import(pkgFull)
	lprog, err := conf.Load()
	if err != nil {
		return nil, err
	}

	pkgInfo := lprog.Package(pkgFull)
	if pkgInfo == nil {
		return nil, errors.New("package was nil")
	}

	return pkgInfo, nil
}

type doc struct {
	PackageName string
	Objects     []obj
	Imports     []string
}

type obj struct {
	InterfaceName string
	Methods       []*method
}
type method struct {
	Name    string
	Params  []*param
	Returns []*param
}

func (m *method) Arglist() string {
	params := make([]string, len(m.Params))
	for i, p := range m.Params {
		params[i] = p.String()
	}
	return strings.Join(params, ", ")
}

func (m *method) ArgCallList() string {
	params := make([]string, len(m.Params))
	for i, p := range m.Params {
		params[i] = p.CallName()
	}
	return strings.Join(params, ", ")
}

func (m *method) ReturnArglist() string {
	params := make([]string, len(m.Returns))
	for i, p := range m.Returns {
		params[i] = p.TypeString()
	}
	if len(m.Returns) > 1 {
		return fmt.Sprintf("(%s)", strings.Join(params, ", "))
	}
	return strings.Join(params, ", ")
}

type param struct {
	Name     string
	Type     string
	Variadic bool
}

func (p param) String() string {
	return fmt.Sprintf("%s %s", p.Name, p.TypeString())
}

func (p param) CallName() string {
	if p.Variadic {
		return p.Name + "..."
	}
	return p.Name
}

func (p param) TypeString() string {
	if p.Variadic {
		return "..." + p.Type[2:]
	}
	return p.Type
}

var templateFuncs = template.FuncMap{
	"Exported": func(s string) string {
		if s == "" {
			return ""
		}
		for _, initialism := range golintInitialisms {
			if strings.ToUpper(s) == initialism {
				return initialism
			}
		}
		return strings.ToUpper(s[0:1]) + s[1:]
	},
}

// pkgPath resolved the full package path from a source directory.
func pkgPath(src string) (string, error) {
	abs, err := filepath.Abs(src)
	if err != nil {
		return "", err
	}
	return stripGopath(abs), nil
}

// stripVendorPath strips the vendor dir prefix from a package path.
// For example we might encounter an absolute path like
// github.com/foo/bar/vendor/github.com/pkg/errors which is resolved
// to github.com/pkg/errors.
func stripVendorPath(p string) string {
	parts := strings.Split(p, "/vendor/")
	if len(parts) == 1 {
		return p
	}
	return strings.TrimLeft(path.Join(parts[1:]...), "/")
}

// stripGopath takes the directory to a package and remove the gopath to get the
// canonical package name.
//
// taken from https://github.com/ernesto-jimenez/gogen
// Copyright (c) 2015 Ernesto Jiménez
func stripGopath(p string) string {
	for _, gopath := range gopaths() {
		p = strings.TrimPrefix(p, path.Join(gopath, "src")+"/")
	}
	return p
}

// gopaths returns a list of GOPATH paths.
//
// See https://github.com/golang/go/blob/master/src/cmd/go/internal/envcmd/env.go#L58
func gopaths() []string {
	return filepath.SplitList(build.Default.GOPATH)
}
