package execution

import (
	"context"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/codera/code-executor/internal/config"
	"github.com/codera/code-executor/internal/domain"
	"github.com/codera/code-executor/internal/sandbox/docker"
)

func getTestConfig() *config.Config {
	return &config.Config{
		CompileTimeout:     5 * time.Second,
		RunTimeout:         2 * time.Second,
		MaxStdoutBytes:     1024,
		MaxStderrBytes:     1024,
		SandboxRuntime:     "docker",
		JavaSandboxImage:   "code-executor-java:latest",
		ExecutionMemory:    "256m",
		ExecutionCPUs:      "1.0",
		ExecutionPidsLimit: 64,
	}
}

func TestJavaExecutor(t *testing.T) {
	cfg := getTestConfig()
	sb := docker.NewRuntime(cfg)
	executor := NewJavaExecutor(cfg, sb)

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

	t.Run("Test 2: Runtime timeout (Infinite loop)", func(t *testing.T) {
		req := domain.ExecutionRequest{
			Language: "java",
			SourceCode: `public class Main {
				public static void main(String[] args) {
					while (true) {}
				}
			}`,
		}

		// Use a tight timeout for this test only
		tightCfg := getTestConfig()
		tightCfg.RunTimeout = 500 * time.Millisecond
		tightSb := docker.NewRuntime(tightCfg)
		tightExecutor := NewJavaExecutor(tightCfg, tightSb)

		start := time.Now()
		res, err := tightExecutor.Execute(context.Background(), req)
		duration := time.Since(start)

		if err != nil {
			t.Fatalf("unexpected platform error: %v", err)
		}
		if res.Status != domain.StatusTimeLimitExceeded {
			t.Errorf("expected TIME_LIMIT_EXCEEDED, got %v", res.Status)
		}
		// Ensure it actually stopped in roughly ~500ms and didn't hang
		if duration > 5*time.Second {
			t.Errorf("execution took too long to timeout: %v", duration)
		}
	})

	t.Run("Test 3: Output limit (Infinite stdout)", func(t *testing.T) {
		req := domain.ExecutionRequest{
			Language: "java",
			SourceCode: `public class Main {
				public static void main(String[] args) {
					while (true) {
						System.out.println("AAAAAAAAAAAAAAAAAAAAAAAA");
					}
				}
			}`,
		}

		// Use a very small output limit
		tightCfg := getTestConfig()
		tightCfg.MaxStdoutBytes = 100
		tightSb := docker.NewRuntime(tightCfg)
		tightExecutor := NewJavaExecutor(tightCfg, tightSb)

		start := time.Now()
		res, err := tightExecutor.Execute(context.Background(), req)
		duration := time.Since(start)

		if err != nil {
			t.Fatalf("unexpected platform error: %v", err)
		}
		if res.Status != domain.StatusOutputLimitExceeded {
			t.Errorf("expected OUTPUT_LIMIT_EXCEEDED, got %v", res.Status)
		}
		if len(res.Stdout) > 100 {
			t.Errorf("stdout exceeded max limit! length: %d", len(res.Stdout))
		}
		// Ensure it didn't hang until a time timeout
		if duration > 5*time.Second {
			t.Errorf("output limit termination took too long: %v", duration)
		}
	})

	t.Run("Test 4: Output limit (Infinite stderr)", func(t *testing.T) {
		req := domain.ExecutionRequest{
			Language: "java",
			SourceCode: `public class Main {
				public static void main(String[] args) {
					while (true) {
						System.err.println("ERROR");
					}
				}
			}`,
		}

		tightCfg := getTestConfig()
		tightCfg.MaxStderrBytes = 50
		tightSb := docker.NewRuntime(tightCfg)
		tightExecutor := NewJavaExecutor(tightCfg, tightSb)

		res, err := tightExecutor.Execute(context.Background(), req)

		if err != nil {
			t.Fatalf("unexpected platform error: %v", err)
		}
		if res.Status != domain.StatusOutputLimitExceeded {
			t.Errorf("expected OUTPUT_LIMIT_EXCEEDED, got %v", res.Status)
		}
		if len(res.Stderr) > 50 {
			t.Errorf("stderr exceeded max limit! length: %d", len(res.Stderr))
		}
	})

	t.Run("Test 5: Compilation error", func(t *testing.T) {
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
	})

	t.Run("Test 6: Runtime error", func(t *testing.T) {
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
	})

	t.Run("Test 7: Fast program doesn't race timeout", func(t *testing.T) {
		req := domain.ExecutionRequest{
			Language: "java",
			SourceCode: `public class Main {
				public static void main(String[] args) {
					// Just exits
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
	})

	t.Run("Test 8: Concurrent Executions Isolation", func(t *testing.T) {
		var wg sync.WaitGroup
		concurrency := 5
		results := make([]domain.ExecutionResult, concurrency)

		for i := 0; i < concurrency; i++ {
			wg.Add(1)
			go func(index int) {
				defer wg.Done()
				req := domain.ExecutionRequest{
					Language: "java",
					SourceCode: `import java.util.Scanner;
					public class Main {
						public static void main(String[] args) {
							Scanner sc = new Scanner(System.in);
							System.out.print("Output " + sc.nextInt());
						}
					}`,
					Input: string(rune('0' + index)),
				}

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

	t.Run("Test 9: Network blocked", func(t *testing.T) {
		req := domain.ExecutionRequest{
			Language: "java",
			SourceCode: `import java.net.URL; import java.net.URLConnection;
			public class Main {
				public static void main(String[] args) {
					try {
						URL url = new URL("http://example.com");
						URLConnection conn = url.openConnection();
						conn.connect();
						System.out.println("CONNECTED");
					} catch (Exception e) {
						System.out.println("FAILED");
					}
				}
			}`,
		}

		res, err := executor.Execute(context.Background(), req)
		if err != nil {
			t.Fatalf("unexpected platform error: %v", err)
		}
		if res.Status != domain.StatusSuccess {
			t.Errorf("expected SUCCESS, got %v (stderr: %v)", res.Status, res.Stderr)
		}
		if !strings.Contains(res.Stdout, "FAILED") {
			t.Errorf("expected network connection to fail, but it succeeded")
		}
	})

	t.Run("Test 10: Read-only filesystem", func(t *testing.T) {
		req := domain.ExecutionRequest{
			Language: "java",
			SourceCode: `import java.io.File; import java.io.FileWriter;
			public class Main {
				public static void main(String[] args) {
					try {
						File file = new File("/tmp/hacked.txt");
						FileWriter writer = new FileWriter(file);
						writer.write("hacked");
						writer.close();
						System.out.println("WROTE");
					} catch (Exception e) {
						System.out.println("FAILED");
					}
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
		if !strings.Contains(res.Stdout, "FAILED") {
			t.Errorf("expected writing outside /workspace to fail due to read-only root fs")
		}
	})
}
