package validators

import (
	"fmt"
	"os"
	"strings"

	"github.com/wasmerio/wasmer-go/wasmer"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
)

type FormatValidator interface {
	Validate(fieldpath []string, validatorContent string, schema *apiextensionsv1.JSONSchemaProps, obj interface{}) error
	ValidateProgram(fieldpath []string, validatorContent string, schema *apiextensionsv1.JSONSchemaProps) error
}

type Converter interface {
	Convert(fieldpath []string, validatorContent string, currentVersion, targetVersion string, currentSchema, targetSchema *apiextensionsv1.JSONSchemaProps, obj interface{}) (interface{}, error)
}

type WasmValidator struct {
	// TODO: Should probably not leave modules running?
	moduleInstances map[string]*wasmer.Instance
}

func NewWasmValidator() *WasmValidator {
	v := &WasmValidator{}
	v.moduleInstances = map[string]*wasmer.Instance{}
	return v
}

func (v *WasmValidator) RegisterModule(filepath string) error {
	b, err := os.ReadFile(filepath)
	engine := wasmer.NewEngine()
	store := wasmer.NewStore(engine)

	// Compiles the module
	module, err := wasmer.NewModule(store, b)
	if err != nil {
		return fmt.Errorf("error creating module: %w", err)
	}

	// Instantiates the module
	importObject := wasmer.NewImportObject()
	instance, err := wasmer.NewInstance(module, importObject)
	if err != nil {
		return fmt.Errorf("error createing module instance: %w", err)
	}
	v.moduleInstances[filepath] = instance
	return nil
}

func (v *WasmValidator) ValidateProgram(fieldpath []string, validatorContent string, schema *apiextensionsv1.JSONSchemaProps) error {
	// TODO
	return nil
}

func (v *WasmValidator) Validate(fieldpath []string, validatorContent string, schema *apiextensionsv1.JSONSchemaProps, obj interface{}) error {
	parts := strings.Split(validatorContent, ":")
	if len(parts) != 2 {
		return fmt.Errorf("expected wasm:<module>:<function-name> but got wasm:%s", validatorContent)
	}
	module := parts[0]
	functionName := parts[1]
	instance := v.moduleInstances[module]

	// Gets the exported function from the WebAssembly instance.
	fn, err := instance.Exports.GetFunction(functionName)
	if err != nil {
		return fmt.Errorf("error getting exported function %s: %w", functionName, err)
	}
	result, err := fn(obj)
	if err != nil {
		return fmt.Errorf("error calling exported function %s: %w", functionName, err)
	}

	if result != nil {
		return fmt.Errorf("validation failed: %w", result)
	}

	return nil
}
