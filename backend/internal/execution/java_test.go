package execution

import (
	"context"
	"strings"
	"sync"
	"testing"

	"github.com/codera/code-executor/internal/domain"
)

func TestJavaExecutor(t *testing.T) {
	executor := NewJavaExecutor()

	t.Run("Test 1: Hello World", func(t *testing.T) {
		req := domain.ExecutionRequest{
			Language: "java",
			SourceCode: `public class Main {
				public static void main(String[] args) {
					System.out.println("Hello World");
				}
			}`,
		}

		res, err := executor.Execute(context.Background(), req)
		if err != nil {
			t.Fatalf("unexpected platform error: %v", err)
		}
		if res.Status != domain.StatusSuccess {
			t.Errorf("expected SUCCESS, got %v", res.Status)
		}
		if strings.TrimSpace(res.Stdout) != "Hello World" {
			t.Errorf("unexpected stdout: %v", res.Stdout)
		}
	})

	t.Run("Test 2: stdin", func(t *testing.T) {
		req := domain.ExecutionRequest{
			Language: "java",
			SourceCode: `import java.util.Scanner;
			public class Main {
				public static void main(String[] args) {
					Scanner sc = new Scanner(System.in);
					int a = sc.nextInt();
					int b = sc.nextInt();
					System.out.println(a + b);
				}
			}`,
			Input: "10 20",
		}

		res, err := executor.Execute(context.Background(), req)
		if err != nil {
			t.Fatalf("unexpected platform error: %v", err)
		}
		if res.Status != domain.StatusSuccess {
			t.Errorf("expected SUCCESS, got %v", res.Status)
		}
		if strings.TrimSpace(res.Stdout) != "30" {
			t.Errorf("unexpected stdout: %v", res.Stdout)
		}
	})

	t.Run("Test 3: Compilation error", func(t *testing.T) {
		req := domain.ExecutionRequest{
			Language: "java",
			SourceCode: `public class Main {
				public static void main(String[] args) {
					System.out.println("Hello") // Missing semicolon
				}
			}`,
		}

		res, err := executor.Execute(context.Background(), req)
		if err != nil {
			t.Fatalf("unexpected platform error: %v", err)
		}
		if res.Status != domain.StatusCompilationError {
			t.Errorf("expected COMPILATION_ERROR, got %v", res.Status)
		}
		if res.Stderr == "" {
			t.Errorf("expected stderr to contain compilation output")
		}
	})

	t.Run("Test 4: Runtime error", func(t *testing.T) {
		req := domain.ExecutionRequest{
			Language: "java",
			SourceCode: `public class Main {
				public static void main(String[] args) {
					int x = 10 / 0;
				}
			}`,
		}

		res, err := executor.Execute(context.Background(), req)
		if err != nil {
			t.Fatalf("unexpected platform error: %v", err)
		}
		if res.Status != domain.StatusRuntimeError {
			t.Errorf("expected RUNTIME_ERROR, got %v", res.Status)
		}
		if !strings.Contains(res.Stderr, "ArithmeticException") {
			t.Errorf("expected stderr to contain ArithmeticException, got %v", res.Stderr)
		}
	})

	t.Run("Test 5: stderr without failure", func(t *testing.T) {
		req := domain.ExecutionRequest{
			Language: "java",
			SourceCode: `public class Main {
				public static void main(String[] args) {
					System.err.println("something happened");
					System.out.println("normal");
				}
			}`,
		}

		res, err := executor.Execute(context.Background(), req)
		if err != nil {
			t.Fatalf("unexpected platform error: %v", err)
		}
		if res.Status != domain.StatusSuccess {
			t.Errorf("expected SUCCESS, got %v", res.Status)
		}
		if strings.TrimSpace(res.Stdout) != "normal" {
			t.Errorf("unexpected stdout: %v", res.Stdout)
		}
		if strings.TrimSpace(res.Stderr) != "something happened" {
			t.Errorf("unexpected stderr: %v", res.Stderr)
		}
	})

	t.Run("Test 6: Multiple executions concurrently", func(t *testing.T) {
		var wg sync.WaitGroup
		concurrency := 10
		results := make([]domain.ExecutionResult, concurrency)

		for i := 0; i < concurrency; i++ {
			wg.Add(1)
			go func(index int) {
				defer wg.Done()
				// Each concurrent request prints its index
				req := domain.ExecutionRequest{
					Language: "java",
					SourceCode: `public class Main {
						public static void main(String[] args) {
							System.out.print("Output " + args[0]);
						}
					}`,
				}
				// Pass index via input, wait we didn't support args, but we support stdin!
				req.SourceCode = `import java.util.Scanner;
				public class Main {
					public static void main(String[] args) {
						Scanner sc = new Scanner(System.in);
						System.out.print("Output " + sc.nextInt());
					}
				}`
				req.Input = string(rune('0' + index)) // Quick int to string since index < 10

				res, _ := executor.Execute(context.Background(), req)
				results[index] = res
			}(i)
		}

		wg.Wait()

		for i := 0; i < concurrency; i++ {
			expected := "Output " + string(rune('0'+i))
			if strings.TrimSpace(results[i].Stdout) != expected {
				t.Errorf("expected %v, got %v", expected, results[i].Stdout)
			}
		}
	})
}
