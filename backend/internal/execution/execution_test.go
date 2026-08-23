package execution

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/codera/code-executor/internal/config"
	"github.com/codera/code-executor/internal/domain"
	"github.com/codera/code-executor/internal/language"
	"github.com/codera/code-executor/internal/language/cpp"
	golang "github.com/codera/code-executor/internal/language/go"
	"github.com/codera/code-executor/internal/language/java"
	"github.com/codera/code-executor/internal/language/python"
	"github.com/codera/code-executor/internal/sandbox"
	"github.com/codera/code-executor/internal/sandbox/docker"
)

func setupService() (*Service, sandbox.Runtime) {
	cfg := &config.Config{
		CompileTimeout:     15 * time.Second,
		RunTimeout:         2 * time.Second,
		MaxStdoutBytes:     1024,
		MaxStderrBytes:     1024,
		SandboxRuntime:     "docker",
	}

	sb := docker.NewRuntime(cfg)
	registry := language.NewMapRegistry()

	javaProfile, _ := sandbox.GetProfileForLanguage("java")
	pythonProfile, _ := sandbox.GetProfileForLanguage("python")
	goProfile, _ := sandbox.GetProfileForLanguage("go")
	cppProfile, _ := sandbox.GetProfileForLanguage("cpp")

	registry.Register(java.NewExecutor(javaProfile))
	registry.Register(python.NewExecutor(pythonProfile))
	registry.Register(golang.NewExecutor(goProfile))
	registry.Register(cpp.NewExecutor(cppProfile))

	return NewService(cfg, registry, sb), sb
}

func TestExecutionService(t *testing.T) {
	svc, _ := setupService()

	t.Run("Unsupported Language", func(t *testing.T) {
		req := domain.ExecutionRequest{
			Language:   "ruby",
			SourceCode: `puts "Hello"`,
		}
		_, err := svc.Execute(context.Background(), req)
		if err == nil || !strings.Contains(err.Error(), "failed to get language executor") {
			t.Errorf("expected unsupported language error, got %v", err)
		}
	})

	t.Run("Java Hello World", func(t *testing.T) {
		req := domain.ExecutionRequest{
			Language: domain.LanguageJava,
			SourceCode: `public class Main {
				public static void main(String[] args) {
					System.out.println("Hello Java");
				}
			}`,
		}
		res, err := svc.Execute(context.Background(), req)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if res.Status != domain.StatusSuccess {
			t.Errorf("expected SUCCESS, got %v", res.Status)
		}
		if strings.TrimSpace(res.Stdout) != "Hello Java" {
			t.Errorf("unexpected stdout: %v", res.Stdout)
		}
	})

	t.Run("Python Hello World", func(t *testing.T) {
		req := domain.ExecutionRequest{
			Language: domain.LanguagePython,
			SourceCode: `print("Hello Python")`,
		}
		res, err := svc.Execute(context.Background(), req)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if res.Status != domain.StatusSuccess {
			t.Errorf("expected SUCCESS, got %v", res.Status)
		}
		if strings.TrimSpace(res.Stdout) != "Hello Python" {
			t.Errorf("unexpected stdout: %v", res.Stdout)
		}
	})

	t.Run("Go Hello World", func(t *testing.T) {
		req := domain.ExecutionRequest{
			Language: domain.LanguageGo,
			SourceCode: `package main
			import "fmt"
			func main() {
				fmt.Println("Hello Go")
			}`,
		}
		res, err := svc.Execute(context.Background(), req)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if res.Status != domain.StatusSuccess {
			t.Errorf("expected SUCCESS, got %v (stderr: %v)", res.Status, res.Stderr)
		}
		if strings.TrimSpace(res.Stdout) != "Hello Go" {
			t.Errorf("unexpected stdout: %v", res.Stdout)
		}
	})

	t.Run("Cpp Hello World", func(t *testing.T) {
		req := domain.ExecutionRequest{
			Language: domain.LanguageCpp,
			SourceCode: `#include <iostream>
			int main() {
				std::cout << "Hello Cpp\n";
				return 0;
			}`,
		}
		res, err := svc.Execute(context.Background(), req)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if res.Status != domain.StatusSuccess {
			t.Errorf("expected SUCCESS, got %v (stderr: %v)", res.Status, res.Stderr)
		}
		if strings.TrimSpace(res.Stdout) != "Hello Cpp" {
			t.Errorf("unexpected stdout: %v", res.Stdout)
		}
	})

	t.Run("Python Syntax Error", func(t *testing.T) {
		req := domain.ExecutionRequest{
			Language: domain.LanguagePython,
			SourceCode: `print("Hello"`, // Missing closing parenthesis
		}
		res, err := svc.Execute(context.Background(), req)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if res.Status != domain.StatusCompilationError {
			t.Errorf("expected COMPILATION_ERROR, got %v", res.Status)
		}
	})
}
