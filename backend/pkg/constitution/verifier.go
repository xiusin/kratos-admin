package constitution

import (
	"bufio"
	"encoding/json"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
)

// antiHallucinationVerifier implements the AntiHallucinationVerifier interface
type antiHallucinationVerifier struct {
	config *Config

	// Index databases
	apiIndex    map[string]*APIReference      // key: serviceName.methodName
	funcIndex   map[string]*FunctionSignature // key: packagePath.functionName
	moduleIndex map[string]bool               // key: modulePath
	configIndex map[string]bool               // key: configKey

	// Mutex for thread-safe access
	mu sync.RWMutex

	// Index status
	indexed bool
}

// NewAntiHallucinationVerifier creates a new anti-hallucination verifier
func NewAntiHallucinationVerifier(config *Config) (AntiHallucinationVerifier, error) {
	if config == nil {
		return nil, fmt.Errorf("config cannot be nil")
	}

	v := &antiHallucinationVerifier{
		config:      config,
		apiIndex:    make(map[string]*APIReference),
		funcIndex:   make(map[string]*FunctionSignature),
		moduleIndex: make(map[string]bool),
		configIndex: make(map[string]bool),
		indexed:     false,
	}

	// Build initial indexes
	if err := v.RebuildIndexes(); err != nil {
		return nil, fmt.Errorf("failed to build indexes: %w", err)
	}

	return v, nil
}

// RebuildIndexes rebuilds all indexes
func (v *antiHallucinationVerifier) RebuildIndexes() error {
	v.mu.Lock()
	defer v.mu.Unlock()

	// Clear existing indexes
	v.apiIndex = make(map[string]*APIReference)
	v.funcIndex = make(map[string]*FunctionSignature)
	v.moduleIndex = make(map[string]bool)
	v.configIndex = make(map[string]bool)

	// Build API index
	if err := v.buildAPIIndex(); err != nil {
		return fmt.Errorf("failed to build API index: %w", err)
	}

	// Build function index
	if err := v.buildFunctionIndex(); err != nil {
		return fmt.Errorf("failed to build function index: %w", err)
	}

	// Build module index
	if err := v.buildModuleIndex(); err != nil {
		return fmt.Errorf("failed to build module index: %w", err)
	}

	// Build config index
	if err := v.buildConfigIndex(); err != nil {
		return fmt.Errorf("failed to build config index: %w", err)
	}

	v.indexed = true
	return nil
}

// VerifyAPIExists verifies that an API exists in Protobuf definitions
func (v *antiHallucinationVerifier) VerifyAPIExists(serviceName, methodName string) (bool, error) {
	v.mu.RLock()
	defer v.mu.RUnlock()

	if !v.indexed {
		return false, fmt.Errorf("indexes not built")
	}

	key := fmt.Sprintf("%s.%s", serviceName, methodName)
	_, exists := v.apiIndex[key]
	return exists, nil
}

// VerifyFunctionExists verifies that a function exists in the codebase
func (v *antiHallucinationVerifier) VerifyFunctionExists(packagePath, functionName string) (bool, error) {
	v.mu.RLock()
	defer v.mu.RUnlock()

	if !v.indexed {
		return false, fmt.Errorf("indexes not built")
	}

	key := fmt.Sprintf("%s.%s", packagePath, functionName)
	_, exists := v.funcIndex[key]
	return exists, nil
}

// VerifyModuleExists verifies that a module exists
func (v *antiHallucinationVerifier) VerifyModuleExists(modulePath string, language string) (bool, error) {
	v.mu.RLock()
	defer v.mu.RUnlock()

	if !v.indexed {
		return false, fmt.Errorf("indexes not built")
	}

	key := fmt.Sprintf("%s:%s", language, modulePath)
	exists := v.moduleIndex[key]
	return exists, nil
}

// VerifyConfigKeyExists verifies that a configuration key exists
func (v *antiHallucinationVerifier) VerifyConfigKeyExists(configKey string) (bool, error) {
	v.mu.RLock()
	defer v.mu.RUnlock()

	if !v.indexed {
		return false, fmt.Errorf("indexes not built")
	}

	exists := v.configIndex[configKey]
	return exists, nil
}

// GetAPIReference retrieves API reference information
func (v *antiHallucinationVerifier) GetAPIReference(serviceName, methodName string) (*APIReference, error) {
	v.mu.RLock()
	defer v.mu.RUnlock()

	if !v.indexed {
		return nil, fmt.Errorf("indexes not built")
	}

	key := fmt.Sprintf("%s.%s", serviceName, methodName)
	ref, exists := v.apiIndex[key]
	if !exists {
		return nil, fmt.Errorf("API not found: %s", key)
	}

	return ref, nil
}

// GetFunctionSignature retrieves function signature information
func (v *antiHallucinationVerifier) GetFunctionSignature(packagePath, functionName string) (*FunctionSignature, error) {
	v.mu.RLock()
	defer v.mu.RUnlock()

	if !v.indexed {
		return nil, fmt.Errorf("indexes not built")
	}

	key := fmt.Sprintf("%s.%s", packagePath, functionName)
	sig, exists := v.funcIndex[key]
	if !exists {
		return nil, fmt.Errorf("function not found: %s", key)
	}

	return sig, nil
}

// buildAPIIndex builds the API index from Protobuf files
func (v *antiHallucinationVerifier) buildAPIIndex() error {
	// Find all .proto files
	protoDir := filepath.Join(v.config.ProjectRoot, "backend", "api", "protos")
	if _, err := os.Stat(protoDir); os.IsNotExist(err) {
		// Proto directory doesn't exist, skip
		return nil
	}

	return filepath.Walk(protoDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() || !strings.HasSuffix(path, ".proto") {
			return nil
		}

		return v.parseProtoFile(path)
	})
}

// parseProtoFile parses a Protobuf file and extracts API definitions
func (v *antiHallucinationVerifier) parseProtoFile(filePath string) error {
	file, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	lineNum := 0

	var currentService string
	var currentDoc strings.Builder

	// Regex patterns
	servicePattern := regexp.MustCompile(`^\s*service\s+(\w+)\s*\{`)
	rpcPattern := regexp.MustCompile(`^\s*rpc\s+(\w+)\s*\(\s*(\w+)\s*\)\s*returns\s*\(\s*(\w+)\s*\)`)
	commentPattern := regexp.MustCompile(`^\s*//\s*(.*)`)

	for scanner.Scan() {
		lineNum++
		line := scanner.Text()

		// Check for service definition
		if matches := servicePattern.FindStringSubmatch(line); matches != nil {
			currentService = matches[1]
			currentDoc.Reset()
			continue
		}

		// Check for RPC method
		if matches := rpcPattern.FindStringSubmatch(line); matches != nil && currentService != "" {
			methodName := matches[1]
			requestType := matches[2]
			responseType := matches[3]

			key := fmt.Sprintf("%s.%s", currentService, methodName)
			v.apiIndex[key] = &APIReference{
				ServiceName:   currentService,
				MethodName:    methodName,
				FilePath:      filePath,
				LineNumber:    lineNum,
				RequestType:   requestType,
				ResponseType:  responseType,
				Documentation: strings.TrimSpace(currentDoc.String()),
			}

			currentDoc.Reset()
			continue
		}

		// Check for comments
		if matches := commentPattern.FindStringSubmatch(line); matches != nil {
			if currentDoc.Len() > 0 {
				currentDoc.WriteString("\n")
			}
			currentDoc.WriteString(matches[1])
		} else if strings.TrimSpace(line) != "" && !strings.HasPrefix(strings.TrimSpace(line), "//") {
			// Non-comment, non-empty line resets documentation
			if !strings.Contains(line, "service") && !strings.Contains(line, "rpc") {
				currentDoc.Reset()
			}
		}
	}

	return scanner.Err()
}

// buildFunctionIndex builds the function index from Go source files
func (v *antiHallucinationVerifier) buildFunctionIndex() error {
	// Find all Go source directories
	backendDir := filepath.Join(v.config.ProjectRoot, "backend")
	if _, err := os.Stat(backendDir); os.IsNotExist(err) {
		// Backend directory doesn't exist, skip
		return nil
	}

	return filepath.Walk(backendDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip vendor, node_modules, and generated directories
		if info.IsDir() {
			name := info.Name()
			if name == "vendor" || name == "node_modules" || name == ".git" || name == "gen" {
				return filepath.SkipDir
			}
			return nil
		}

		if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}

		return v.parseGoFile(path)
	})
}

// parseGoFile parses a Go file and extracts function signatures
func (v *antiHallucinationVerifier) parseGoFile(filePath string) error {
	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, filePath, nil, parser.ParseComments)
	if err != nil {
		// Skip files with parse errors
		return nil
	}

	packagePath := node.Name.Name

	// Extract package path from file path
	relPath, err := filepath.Rel(filepath.Join(v.config.ProjectRoot, "backend"), filePath)
	if err == nil {
		dir := filepath.Dir(relPath)
		if dir != "." {
			packagePath = strings.ReplaceAll(dir, string(filepath.Separator), "/")
		}
	}

	// Visit all declarations
	for _, decl := range node.Decls {
		funcDecl, ok := decl.(*ast.FuncDecl)
		if !ok {
			continue
		}

		// Skip unexported functions
		if !funcDecl.Name.IsExported() {
			continue
		}

		// Extract function signature
		sig := v.extractFunctionSignature(funcDecl, fset, filePath, packagePath)
		if sig != nil {
			key := fmt.Sprintf("%s.%s", packagePath, sig.FunctionName)
			v.funcIndex[key] = sig
		}
	}

	return nil
}

// extractFunctionSignature extracts function signature from AST
func (v *antiHallucinationVerifier) extractFunctionSignature(funcDecl *ast.FuncDecl, fset *token.FileSet, filePath, packagePath string) *FunctionSignature {
	sig := &FunctionSignature{
		PackagePath:  packagePath,
		FunctionName: funcDecl.Name.Name,
		FilePath:     filePath,
		LineNumber:   fset.Position(funcDecl.Pos()).Line,
		Parameters:   []string{},
		ReturnTypes:  []string{},
	}

	// Extract documentation
	if funcDecl.Doc != nil {
		var doc strings.Builder
		for _, comment := range funcDecl.Doc.List {
			text := strings.TrimPrefix(comment.Text, "//")
			text = strings.TrimPrefix(text, "/*")
			text = strings.TrimSuffix(text, "*/")
			text = strings.TrimSpace(text)
			if doc.Len() > 0 {
				doc.WriteString("\n")
			}
			doc.WriteString(text)
		}
		sig.Documentation = doc.String()
	}

	// Extract parameters
	if funcDecl.Type.Params != nil {
		for _, param := range funcDecl.Type.Params.List {
			paramType := v.exprToString(param.Type)
			for _, name := range param.Names {
				sig.Parameters = append(sig.Parameters, fmt.Sprintf("%s %s", name.Name, paramType))
			}
			if len(param.Names) == 0 {
				sig.Parameters = append(sig.Parameters, paramType)
			}
		}
	}

	// Extract return types
	if funcDecl.Type.Results != nil {
		for _, result := range funcDecl.Type.Results.List {
			resultType := v.exprToString(result.Type)
			sig.ReturnTypes = append(sig.ReturnTypes, resultType)
		}
	}

	return sig
}

// exprToString converts an AST expression to string
func (v *antiHallucinationVerifier) exprToString(expr ast.Expr) string {
	switch t := expr.(type) {
	case *ast.Ident:
		return t.Name
	case *ast.StarExpr:
		return "*" + v.exprToString(t.X)
	case *ast.SelectorExpr:
		return v.exprToString(t.X) + "." + t.Sel.Name
	case *ast.ArrayType:
		return "[]" + v.exprToString(t.Elt)
	case *ast.MapType:
		return "map[" + v.exprToString(t.Key) + "]" + v.exprToString(t.Value)
	case *ast.InterfaceType:
		return "interface{}"
	case *ast.ChanType:
		return "chan " + v.exprToString(t.Value)
	default:
		return "unknown"
	}
}

// buildModuleIndex builds the module index from go.mod and package.json
func (v *antiHallucinationVerifier) buildModuleIndex() error {
	// Build Go module index
	if err := v.buildGoModuleIndex(); err != nil {
		return err
	}

	// Build NPM module index
	if err := v.buildNPMModuleIndex(); err != nil {
		return err
	}

	return nil
}

// buildGoModuleIndex builds the Go module index from go.mod
func (v *antiHallucinationVerifier) buildGoModuleIndex() error {
	goModPath := filepath.Join(v.config.ProjectRoot, "backend", "go.mod")
	if _, err := os.Stat(goModPath); os.IsNotExist(err) {
		return nil
	}

	file, err := os.Open(goModPath)
	if err != nil {
		return err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	requirePattern := regexp.MustCompile(`^\s*([^\s]+)\s+v`)

	for scanner.Scan() {
		line := scanner.Text()
		if matches := requirePattern.FindStringSubmatch(line); matches != nil {
			modulePath := matches[1]
			key := fmt.Sprintf("go:%s", modulePath)
			v.moduleIndex[key] = true
		}
	}

	return scanner.Err()
}

// buildNPMModuleIndex builds the NPM module index from package.json
func (v *antiHallucinationVerifier) buildNPMModuleIndex() error {
	packageJSONPath := filepath.Join(v.config.ProjectRoot, "frontend", "package.json")
	if _, err := os.Stat(packageJSONPath); os.IsNotExist(err) {
		return nil
	}

	data, err := os.ReadFile(packageJSONPath)
	if err != nil {
		return err
	}

	var pkg struct {
		Dependencies    map[string]string `json:"dependencies"`
		DevDependencies map[string]string `json:"devDependencies"`
	}

	if err := json.Unmarshal(data, &pkg); err != nil {
		return err
	}

	// Add all dependencies
	for moduleName := range pkg.Dependencies {
		key := fmt.Sprintf("npm:%s", moduleName)
		v.moduleIndex[key] = true
	}

	for moduleName := range pkg.DevDependencies {
		key := fmt.Sprintf("npm:%s", moduleName)
		v.moduleIndex[key] = true
	}

	return nil
}

// buildConfigIndex builds the configuration key index
func (v *antiHallucinationVerifier) buildConfigIndex() error {
	// Find all YAML config files
	configDir := filepath.Join(v.config.ProjectRoot, "backend", "app")
	if _, err := os.Stat(configDir); os.IsNotExist(err) {
		return nil
	}

	return filepath.Walk(configDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		if !strings.HasSuffix(path, ".yaml") && !strings.HasSuffix(path, ".yml") {
			return nil
		}

		return v.parseConfigFile(path)
	})
}

// parseConfigFile parses a YAML config file and extracts keys
func (v *antiHallucinationVerifier) parseConfigFile(filePath string) error {
	file, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	keyPattern := regexp.MustCompile(`^(\s*)([a-zA-Z_][a-zA-Z0-9_]*):`)

	var keyStack []string
	var indentStack []int

	for scanner.Scan() {
		line := scanner.Text()

		// Skip comments and empty lines
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}

		matches := keyPattern.FindStringSubmatch(line)
		if matches == nil {
			continue
		}

		indent := len(matches[1])
		key := matches[2]

		// Adjust key stack based on indentation
		for len(indentStack) > 0 && indentStack[len(indentStack)-1] >= indent {
			keyStack = keyStack[:len(keyStack)-1]
			indentStack = indentStack[:len(indentStack)-1]
		}

		// Add current key
		keyStack = append(keyStack, key)
		indentStack = append(indentStack, indent)

		// Build full key path
		fullKey := strings.Join(keyStack, ".")
		v.configIndex[fullKey] = true
	}

	return scanner.Err()
}
