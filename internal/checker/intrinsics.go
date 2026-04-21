package checker

import (
	"errors"
	"slices"
	"strings"

	"github.com/microsoft/typescript-go/internal/ast"
	intrinsicdsl "github.com/microsoft/typescript-go/internal/checker/intrinsicdsl"
	"github.com/microsoft/typescript-go/internal/diagnostics"
	"github.com/microsoft/typescript-go/internal/jsnum"
)

func (c *Checker) resolveIntrinsicArg(t *Type) *Type {
	if t.flags&TypeFlagsStringLiteral != 0 {
		return t
	}
	src := c.applySourceOf(t)
	if src.flags&TypeFlagsStringLiteral != 0 {
		return src
	}
	return t
}

func (c *Checker) applySourceOf(t *Type) *Type {
	sigs := c.getSignaturesOfType(t, SignatureKindCall)
	if len(sigs) == 0 {
		return c.stringType
	}
	decl := sigs[0].declaration
	if decl == nil {
		return c.stringType
	}
	if decl.Kind != ast.KindArrowFunction && decl.Kind != ast.KindFunctionExpression {
		return c.stringType
	}
	sourceFile := ast.GetSourceFileOfNode(decl)
	if sourceFile == nil {
		return c.stringType
	}

	mainSource := extractNodeSource(decl, sourceFile)
	if mainSource == "" {
		return c.stringType
	}

	topLevelDecls := c.buildTopLevelDeclMap(sourceFile)
	paramNames := c.collectParamNames(decl)

	deps, err := c.collectIntrinsicDeps(decl, topLevelDecls, paramNames)
	if err != "" {
		if c.currentNode != nil {
			c.error(c.currentNode, diagnostics.Intrinsic_type_parse_error_Colon_0, err)
		}
		return c.errorType
	}

	if len(deps) == 0 {
		return c.getStringLiteralType(mainSource)
	}

	ordered := c.topoSortDeps(deps)
	var preamble strings.Builder
	for _, name := range ordered {
		dep := deps[name]
		preamble.WriteString("let ")
		preamble.WriteString(name)
		preamble.WriteString(" = ")
		preamble.WriteString(dep.source)
		preamble.WriteString(";\n")
	}

	return c.getStringLiteralType(preamble.String() + mainSource)
}

type intrinsicDep struct {
	source string
	deps   []string
}

type topLevelDeclInfo struct {
	node       *ast.Node
	isFunction bool
	sourceText string
	sourceFile *ast.SourceFile
}

func extractNodeSource(node *ast.Node, sf *ast.SourceFile) string {
	text := sf.Text()
	pos := node.Pos()
	end := node.End()
	if pos < 0 || end > len(text) || pos >= end {
		return ""
	}
	return strings.TrimSpace(text[pos:end])
}

// buildTopLevelDeclMap collects const variable declarations and resolved named
// imports from the source file's top-level statements.
func (c *Checker) buildTopLevelDeclMap(sf *ast.SourceFile) map[string]*topLevelDeclInfo {
	decls := make(map[string]*topLevelDeclInfo)
	if sf.Statements == nil {
		return decls
	}
	for _, stmt := range sf.Statements.Nodes {
		switch stmt.Kind {
		case ast.KindVariableStatement:
			c.collectConstDecls(stmt, sf, decls)
		case ast.KindImportDeclaration, ast.KindJSImportDeclaration:
			c.collectImportedDecls(stmt, decls)
		}
	}
	return decls
}

func (c *Checker) collectConstDecls(stmt *ast.Node, sf *ast.SourceFile, decls map[string]*topLevelDeclInfo) {
	declList := stmt.AsVariableStatement().DeclarationList
	if declList == nil {
		return
	}
	if declList.Flags&ast.NodeFlagsConst == 0 {
		return
	}
	if declList.AsVariableDeclarationList().Declarations == nil {
		return
	}
	for _, d := range declList.AsVariableDeclarationList().Declarations.Nodes {
		vd := d.AsVariableDeclaration()
		name := vd.Name()
		if name == nil || name.Kind != ast.KindIdentifier {
			continue
		}
		init := vd.Initializer
		if init == nil {
			continue
		}
		src := extractNodeSource(init, sf)
		if src == "" {
			continue
		}
		isFn := init.Kind == ast.KindArrowFunction || init.Kind == ast.KindFunctionExpression
		decls[name.Text()] = &topLevelDeclInfo{node: init, isFunction: isFn, sourceText: src, sourceFile: sf}
	}
}

func (c *Checker) collectImportedDecls(stmt *ast.Node, decls map[string]*topLevelDeclInfo) {
	importDecl := stmt.AsImportDeclaration()
	if importDecl.ImportClause == nil {
		return
	}
	clause := importDecl.ImportClause.AsImportClause()
	if clause.NamedBindings == nil || clause.NamedBindings.Kind != ast.KindNamedImports {
		return
	}
	namedImports := clause.NamedBindings.AsNamedImports()
	if namedImports.Elements == nil {
		return
	}
	for _, specNode := range namedImports.Elements.Nodes {
		spec := specNode.AsImportSpecifier()
		if spec.IsTypeOnly {
			continue
		}
		localName := spec.Name()
		if localName == nil {
			continue
		}
		info := c.resolveImportSpecifierDecl(specNode)
		if info == nil {
			continue
		}
		decls[localName.Text()] = info
	}
}

func (c *Checker) resolveImportSpecifierDecl(specNode *ast.Node) *topLevelDeclInfo {
	sym := specNode.Symbol()
	if sym == nil {
		return nil
	}
	if sym.Flags&ast.SymbolFlagsAlias != 0 {
		sym = c.resolveAlias(sym)
	}
	if sym == nil || sym == c.unknownSymbol {
		return nil
	}
	valueDecl := sym.ValueDeclaration
	if valueDecl == nil || valueDecl.Kind != ast.KindVariableDeclaration {
		return nil
	}
	if ast.GetCombinedNodeFlags(valueDecl)&ast.NodeFlagsConst == 0 {
		return nil
	}
	vd := valueDecl.AsVariableDeclaration()
	init := vd.Initializer
	if init == nil {
		return nil
	}
	sf := ast.GetSourceFileOfNode(valueDecl)
	if sf == nil {
		return nil
	}
	src := extractNodeSource(init, sf)
	if src == "" {
		return nil
	}
	isFn := init.Kind == ast.KindArrowFunction || init.Kind == ast.KindFunctionExpression
	return &topLevelDeclInfo{node: init, isFunction: isFn, sourceText: src, sourceFile: sf}
}

func (c *Checker) collectParamNames(decl *ast.Node) map[string]bool {
	params := make(map[string]bool)
	paramList := decl.ParameterList()
	if paramList == nil {
		return params
	}
	for _, p := range paramList.Nodes {
		name := p.AsParameterDeclaration().Name()
		if name != nil && name.Kind == ast.KindIdentifier {
			params[name.Text()] = true
		}
	}
	return params
}

func (c *Checker) collectIntrinsicDeps(
	decl *ast.Node,
	topLevelDecls map[string]*topLevelDeclInfo,
	exclude map[string]bool,
) (map[string]*intrinsicDep, string) {
	deps := make(map[string]*intrinsicDep)
	var errMsg string

	declMapCache := map[*ast.SourceFile]map[string]*topLevelDeclInfo{}

	var collect func(node *ast.Node, localExclude map[string]bool, declMap map[string]*topLevelDeclInfo) string
	collect = func(node *ast.Node, localExclude map[string]bool, declMap map[string]*topLevelDeclInfo) string {
		if errMsg != "" {
			return errMsg
		}

		freeIdents := make(map[string]bool)
		c.collectFreeIdentifiers(node, localExclude, freeIdents)

		sortedIdents := make([]string, 0, len(freeIdents))
		for name := range freeIdents {
			sortedIdents = append(sortedIdents, name)
		}
		slices.Sort(sortedIdents)

		for _, name := range sortedIdents {
			if deps[name] != nil {
				continue
			}
			info, ok := declMap[name]
			if !ok {
				continue
			}

			testSrc := "let " + name + " = " + info.sourceText + ";\n(__unused__: any) => 0"
			if _, err := intrinsicdsl.ParseProgram(testSrc); err != nil {
				continue
			}

			dep := &intrinsicDep{source: info.sourceText}
			deps[name] = dep

			depDeclMap := declMap
			if info.sourceFile != nil {
				sf := info.sourceFile
				if cached, ok := declMapCache[sf]; ok {
					depDeclMap = cached
				} else {
					built := c.buildTopLevelDeclMap(sf)
					declMapCache[sf] = built
					depDeclMap = built
				}
			}

			depExclude := localExclude
			if info.isFunction {
				depExclude = make(map[string]bool)
				for k, v := range localExclude {
					depExclude[k] = v
				}
				depParams := c.collectParamNames(info.node)
				for k, v := range depParams {
					depExclude[k] = v
				}
			}
			if e := collect(info.node, depExclude, depDeclMap); e != "" {
				return e
			}

			subFree := make(map[string]bool)
			c.collectFreeIdentifiers(info.node, depExclude, subFree)
			sortedSub := make([]string, 0, len(subFree))
			for subName := range subFree {
				sortedSub = append(sortedSub, subName)
			}
			slices.Sort(sortedSub)
			for _, subName := range sortedSub {
				if deps[subName] != nil {
					dep.deps = append(dep.deps, subName)
				}
			}
		}
		return ""
	}

	errMsg = collect(decl, exclude, topLevelDecls)
	if errMsg != "" {
		return nil, errMsg
	}
	return deps, ""
}

func (c *Checker) collectFreeIdentifiers(node *ast.Node, exclude map[string]bool, result map[string]bool) {
	c.collectFreeIdentsImpl(node, exclude, nil, result)
}

func (c *Checker) collectFreeIdentsImpl(node *ast.Node, exclude map[string]bool, localNames map[string]bool, result map[string]bool) {
	if localNames == nil {
		localNames = make(map[string]bool)
	}

	var walk ast.Visitor
	walk = func(n *ast.Node) bool {
		if n == nil {
			return false
		}
		switch n.Kind {
		case ast.KindIdentifier:
			name := n.Text()
			if !exclude[name] && !localNames[name] {
				result[name] = true
			}
		case ast.KindVariableDeclaration:
			vd := n.AsVariableDeclaration()
			if vd.Name() != nil && vd.Name().Kind == ast.KindIdentifier {
				localNames[vd.Name().Text()] = true
			}
			if vd.Initializer != nil {
				vd.Initializer.ForEachChild(walk)
			}
			return false
		case ast.KindArrowFunction, ast.KindFunctionExpression:
			innerExclude := make(map[string]bool)
			for k := range exclude {
				innerExclude[k] = true
			}
			for k := range localNames {
				innerExclude[k] = true
			}
			paramList := n.ParameterList()
			if paramList != nil {
				for _, p := range paramList.Nodes {
					pName := p.AsParameterDeclaration().Name()
					if pName != nil && pName.Kind == ast.KindIdentifier {
						innerExclude[pName.Text()] = true
					}
				}
			}
			body := n.BodyData().Body
			if body != nil {
				c.collectFreeIdentsImpl(body, innerExclude, nil, result)
			}
			return false
		case ast.KindFunctionDeclaration:
			fd := n.AsFunctionDeclaration()
			if fd.Name() != nil {
				localNames[fd.Name().Text()] = true
			}
			return false
		case ast.KindParameter:
			return false
		}
		n.ForEachChild(walk)
		return false
	}
	node.ForEachChild(walk)
}

func (c *Checker) topoSortDeps(deps map[string]*intrinsicDep) []string {
	visited := make(map[string]bool)
	var order []string

	var visit func(name string)
	visit = func(name string) {
		if visited[name] {
			return
		}
		visited[name] = true
		dep := deps[name]
		if dep != nil {
			for _, subName := range dep.deps {
				visit(subName)
			}
		}
		order = append(order, name)
	}

	names := make([]string, 0, len(deps))
	for name := range deps {
		names = append(names, name)
	}
	slices.Sort(names)
	for _, name := range names {
		visit(name)
	}
	return order
}

func (c *Checker) typeToDslValue(t *Type) (intrinsicdsl.Value, bool) {
	switch {
	case t.flags&TypeFlagsStringLiteral != 0:
		return intrinsicdsl.StrVal(getStringLiteralValue(t)), true
	case t.flags&TypeFlagsNumberLiteral != 0:
		return intrinsicdsl.NumVal(float64(getNumberLiteralValue(t))), true
	case t == c.trueType || t == c.regularTrueType:
		return intrinsicdsl.BoolVal(true), true
	case t == c.falseType || t == c.regularFalseType:
		return intrinsicdsl.BoolVal(false), true
	case t == c.nullType:
		return intrinsicdsl.Null, true
	case t == c.undefinedType:
		return intrinsicdsl.Undefined, true
	case t.flags&TypeFlagsString != 0:
		return intrinsicdsl.StringType, true
	case t.flags&TypeFlagsNumber != 0:
		return intrinsicdsl.NumberType, true
	case t.flags&TypeFlagsBoolean != 0:
		return intrinsicdsl.BooleanType, true
	case isTupleType(t):
		elemTypes := c.getElementTypes(t)
		elems := make([]intrinsicdsl.Value, len(elemTypes))
		for i, et := range elemTypes {
			v, ok := c.typeToDslValue(et)
			if !ok {
				return intrinsicdsl.Value{}, false
			}
			elems[i] = v
		}
		return intrinsicdsl.TupleVal(elems...), true
	case t.flags&TypeFlagsObject != 0:
		props := c.getPropertiesOfType(t)
		m := intrinsicdsl.NewOrderedMap()
		for _, prop := range props {
			val, ok := c.typeToDslValue(c.getTypeOfSymbol(prop))
			if !ok {
				return intrinsicdsl.Value{}, false
			}
			m.Set(prop.Name, val)
		}
		return intrinsicdsl.ObjectVal(m), true
	}
	return intrinsicdsl.Value{}, false
}

func (c *Checker) dslValueToType(v intrinsicdsl.Value) *Type {
	switch v.Kind {
	case intrinsicdsl.KindNumber:
		return c.getNumberLiteralType(jsnum.Number(v.Num))
	case intrinsicdsl.KindString:
		return c.getStringLiteralType(v.Str)
	case intrinsicdsl.KindBoolean:
		if v.Bool {
			return c.trueType
		}
		return c.falseType
	case intrinsicdsl.KindNull:
		return c.nullType
	case intrinsicdsl.KindUndefined:
		return c.undefinedType
	case intrinsicdsl.KindTypeSentinel:
		if len(v.Str) > 6 && v.Str[:6] == "error:" {
			if c.currentNode != nil {
				c.error(c.currentNode, diagnostics.Intrinsic_type_validation_error_Colon_0, v.Str[6:])
			}
			return c.errorType
		}
		switch v.Str {
		case "never":
			return c.neverType
		case "unknown":
			return c.unknownType
		case "void":
			return c.voidType
		case "number":
			return c.numberType
		case "string":
			return c.stringType
		case "boolean":
			return c.booleanType
		default:
			return c.errorType
		}
	case intrinsicdsl.KindTuple:
		elemTypes := make([]*Type, len(v.Elems))
		for i, e := range v.Elems {
			elemTypes[i] = c.dslValueToType(e)
		}
		elementFlags := make([]TupleElementInfo, len(elemTypes))
		for i := range elementFlags {
			elementFlags[i] = TupleElementInfo{flags: ElementFlagsRequired}
		}
		return c.createTupleTypeEx(elemTypes, elementFlags, false /*readonly*/)
	case intrinsicdsl.KindObject:
		members := make(ast.SymbolTable)
		orderedProps := make([]*ast.Symbol, 0, v.Props.Size())
		for key, propVal := range v.Props.Entries() {
			prop := c.newSymbol(ast.SymbolFlagsProperty, key)
			c.valueSymbolLinks.Get(prop).resolvedType = c.dslValueToType(propVal)
			members[prop.Name] = prop
			orderedProps = append(orderedProps, prop)
		}
		t := c.newObjectType(ObjectFlagsAnonymous, nil)
		t.objectFlags |= ObjectFlagsMembersResolved
		data := t.AsStructuredType()
		data.members = members
		data.properties = orderedProps
		return t
	case intrinsicdsl.KindFunction:
		return c.neverType
	}
	return c.neverType
}

func (c *Checker) applyCustomIntrinsic(symbol *ast.Symbol, type1 *Type, type2 *Type) *Type {
	if type1.flags&TypeFlagsStringLiteral != 0 {
		if c.currentNode != nil {
			c.error(c.currentNode, diagnostics.Intrinsic_type_requires_a_function_type)
		}
		return c.errorType
	}

	source := c.applySourceOf(type1)
	if source == c.errorType {
		return c.errorType
	}
	if source.flags&TypeFlagsStringLiteral == 0 {
		return c.getDeferredIntrinsicType(symbol, []*Type{type1, type2})
	}
	funBody := getStringLiteralValue(source)

	if !isTupleType(type2) {
		return c.getDeferredIntrinsicType(symbol, []*Type{type1, type2})
	}

	elemTypes := c.getElementTypes(type2)
	dslArgs := make([]intrinsicdsl.Value, len(elemTypes))
	for i, et := range elemTypes {
		v, ok := c.typeToDslValue(et)
		if !ok {
			return c.getDeferredIntrinsicType(symbol, []*Type{type1, type2})
		}
		dslArgs[i] = v
	}

	cacheKey := funBody + "\x00"
	for i, a := range dslArgs {
		if i > 0 {
			cacheKey += "\x00"
		}
		cacheKey += intrinsicdsl.CacheKey(a)
	}
	if c.intrinsicResultCache == nil {
		c.intrinsicResultCache = make(map[string]*Type)
	}
	if cached, ok := c.intrinsicResultCache[cacheKey]; ok {
		return cached
	}

	astCache := c.getIntrinsicAstCache()
	program, ok := astCache[funBody]
	if !ok {
		var err error
		program, err = intrinsicdsl.ParseProgram(funBody)
		if err != nil {
			if c.currentNode != nil {
				c.error(c.currentNode, diagnostics.Intrinsic_type_parse_error_Colon_0, err.Error())
			}
			c.intrinsicResultCache[cacheKey] = c.errorType
			return c.errorType
		}
		astCache[funBody] = program
	}

	result, err := intrinsicdsl.Run(program, dslArgs, intrinsicdsl.DefaultBudget)
	var resultType *Type
	if err != nil {
		if c.currentNode != nil {
			if errors.Is(err, intrinsicdsl.ErrBudgetExceeded) || errors.Is(err, intrinsicdsl.ErrMemoryExceeded) {
				c.error(c.currentNode, diagnostics.Intrinsic_type_evaluation_budget_exceeded)
			} else {
				c.error(c.currentNode, diagnostics.Intrinsic_type_runtime_error_Colon_0, err.Error())
			}
		}
		resultType = c.errorType
	} else {
		resultType = c.dslValueToType(result.Value)
	}

	c.intrinsicResultCache[cacheKey] = resultType
	return resultType
}

func (c *Checker) getIntrinsicAstCache() map[string]*intrinsicdsl.Node {
	if c.intrinsicAstCache == nil {
		m := make(map[string]*intrinsicdsl.Node)
		c.intrinsicAstCache = m
		return m
	}
	return c.intrinsicAstCache.(map[string]*intrinsicdsl.Node)
}

func (c *Checker) getDeferredIntrinsicType(symbol *ast.Symbol, types []*Type) *Type {
	var b keyBuilder
	b.writeSymbol(symbol)
	for _, t := range types {
		b.writeType(t)
	}
	key := b.hash()
	if c.deferredIntrinsicTypes == nil {
		c.deferredIntrinsicTypes = make(map[CacheHashKey]*Type)
	}
	if result, ok := c.deferredIntrinsicTypes[key]; ok {
		return result
	}
	result := c.newDeferredIntrinsicType(symbol, types)
	c.deferredIntrinsicTypes[key] = result
	return result
}

func (c *Checker) newDeferredIntrinsicType(symbol *ast.Symbol, types []*Type) *Type {
	data := &DeferredIntrinsicType{}
	data.innerTypes = types
	t := c.newType(TypeFlagsDeferredIntrinsic, ObjectFlagsNone, data)
	t.symbol = symbol
	return t
}
