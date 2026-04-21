package main

import (
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/microsoft/typescript-go/internal/ast"
	"github.com/microsoft/typescript-go/internal/checker"
	"github.com/microsoft/typescript-go/internal/compiler"
	"github.com/microsoft/typescript-go/internal/execute/tsc"
	"github.com/microsoft/typescript-go/internal/scanner"
	"github.com/microsoft/typescript-go/internal/tsoptions"
	"github.com/microsoft/typescript-go/internal/tspath"
)

func shouldDumpVarType(chk *checker.Checker, t *checker.Type) bool {
	const wideUnionThreshold = 4
	if t.IsUnion() && len(t.Types()) > wideUnionThreshold {
		return false
	}
	if len(chk.GetSignaturesOfType(t, checker.SignatureKindCall)) > 0 {
		return false
	}
	return true
}

func emitBindingName(
	chk *checker.Checker,
	file *ast.SourceFile,
	decl *ast.Node,
	nameNode *ast.Node,
	record func(int, string),
) {
	lineOf := func(n *ast.Node) int {
		pos := scanner.SkipTrivia(file.Text(), n.Pos())
		line, _ := scanner.GetECMALineAndUTF16CharacterOfPosition(file, pos)
		return line + 1
	}
	emit := func(idNode *ast.Node) {
		t := chk.GetTypeAtLocation(idNode)
		if t == nil || !shouldDumpVarType(chk, t) {
			return
		}
		record(lineOf(decl), fmt.Sprintf("const %s: %s", idNode.Text(), chk.TypeToString(t)))
	}
	if ast.IsIdentifier(nameNode) {
		emit(nameNode)
		return
	}
	if !ast.IsBindingPattern(nameNode) {
		return
	}
	var walk func(n *ast.Node)
	walk = func(n *ast.Node) {
		n.ForEachChild(func(child *ast.Node) bool {
			if ast.IsBindingElement(child) {
				childName := child.Name()
				if childName == nil {
					return false
				}
				if ast.IsIdentifier(childName) {
					emit(childName)
				} else if ast.IsBindingPattern(childName) {
					walk(childName)
				}
			}
			return false
		})
	}
	walk(nameNode)
}

func runDumpTypes(args []string) int {
	sys := newSystem()
	out := sys.Writer()

	commandLine := tsoptions.ParseCommandLine(args, sys)
	if len(commandLine.Errors) > 0 {
		for _, e := range commandLine.Errors {
			fmt.Fprintln(out, e.String())
		}
		return int(tsc.ExitStatusDiagnosticsPresent_OutputsSkipped)
	}

	configForCompilation := commandLine
	if len(commandLine.FileNames()) == 0 {
		cfg := tspath.CombinePaths(sys.GetCurrentDirectory(), "tsconfig.json")
		if sys.FS().FileExists(cfg) {
			extendedConfigCache := &tsc.ExtendedConfigCache{}
			parseResult, errors := tsoptions.GetParsedCommandLineOfConfigFile(
				cfg,
				commandLine.CompilerOptions(),
				nil,
				sys,
				extendedConfigCache,
			)
			if len(errors) > 0 {
				for _, e := range errors {
					fmt.Fprintln(out, e.String())
				}
				return int(tsc.ExitStatusDiagnosticsPresent_OutputsSkipped)
			}
			configForCompilation = parseResult
		}
	}

	host := compiler.NewCachedFSCompilerHost(
		sys.GetCurrentDirectory(),
		sys.FS(),
		sys.DefaultLibraryPath(),
		&tsc.ExtendedConfigCache{},
		nil,
	)
	program := compiler.NewProgram(compiler.ProgramOptions{
		Config: configForCompilation,
		Host:   host,
	})

	ctx := context.Background()
	defaultLibPath := sys.DefaultLibraryPath()

	for _, file := range program.SourceFiles() {
		name := file.FileName()
		// Skip embedded default lib files.
		if strings.HasPrefix(name, "bundled:///") ||
			(defaultLibPath != "" && strings.HasPrefix(name, defaultLibPath)) {
			continue
		}
		writeAliases(out, ctx, program, file)
	}

	return 0
}

func writeAliases(w io.Writer, ctx context.Context, program *compiler.Program, file *ast.SourceFile) {
	var lines []string
	lineOf := func(n *ast.Node) int {
		pos := scanner.SkipTrivia(file.Text(), n.Pos())
		line, _ := scanner.GetECMALineAndUTF16CharacterOfPosition(file, pos)
		return line + 1
	}
	record := func(line int, text string) {
		lines = append(lines, fmt.Sprintf("  L%d: %s", line, text))
	}

	for _, stmt := range file.Statements.Nodes {
		switch {
		case ast.IsTypeAliasDeclaration(stmt):
			chk, done := program.GetTypeCheckerForFile(ctx, file)
			aliasName := ""
			if nameNode := stmt.Name(); nameNode != nil {
				aliasName = nameNode.Text()
			}
			sym := chk.GetSymbolAtLocation(stmt.Name())
			t := chk.GetDeclaredTypeOfSymbol(sym)
			if t == nil {
				t = chk.GetTypeAtLocation(stmt.AsNode())
			}
			record(lineOf(stmt), fmt.Sprintf("type %s = %s", aliasName, chk.TypeToString(t)))
			done()

		case ast.IsVariableStatement(stmt):
			declList := stmt.AsVariableStatement().DeclarationList.AsVariableDeclarationList()
			for _, decl := range declList.Declarations.Nodes {
				nameNode := decl.Name()
				if nameNode == nil {
					continue
				}
				chk, done := program.GetTypeCheckerForFile(ctx, file)
				emitBindingName(chk, file, decl, nameNode, record)
				done()
			}
		}
	}

	if len(lines) == 0 {
		return
	}
	fmt.Fprintf(w, "=== %s ===\n", file.FileName())
	for _, l := range lines {
		fmt.Fprintln(w, l)
	}
}
