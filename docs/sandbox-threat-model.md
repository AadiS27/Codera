# Sandbox Threat Model

## 1. Trust Boundaries

### 1.1 Trusted Components
* **Host Operating System (Linux kernel)**: We assume the kernel is fundamentally secure and patched against known 0-day exploits. We rely on its isolation features (namespaces, cgroups, seccomp).
* **Docker Daemon / Container Runtime**: We assume the container runtime correctly enforces configured namespaces, mounts, and capabilities.
* **Worker Process**: The Go process managing job lifecycles and launching the sandbox is trusted.

### 1.2 Untrusted Components
* **User Code**: Any source code, input data, or binaries submitted for execution. It must be treated as actively malicious.

## 2. In-Scope Attacks (Defended Against)

* **Resource Exhaustion (Denial of Service)**:
  * **Memory Bombs**: Rapidly allocating memory to induce host OOM. (Defended by cgroups `memory` limit).
  * **CPU Abuse**: Infinite loops spinning CPU cycles. (Defended by cgroups `cpuset` quota and execution timeout).
  * **Fork Bombs**: Rapidly forking thousands of processes to exhaust host process tables. (Defended by cgroups `pids` limit).
  * **Disk Exhaustion**: Writing massively large files to fill host disk space. (Defended by read-only root FS and fixed-size `tmpfs`).
  * **Output Flooding**: Printing megabytes of stdout/stderr to crash the worker process parsing it. (Defended by bounded process output readers).
  
* **Privilege Escalation & Host Access**:
  * **Filesystem Escapes**: Attempting to read `/etc/passwd`, Docker socket, or secrets. (Defended by empty containers, isolated mount namespace, and no host volume mounts).
  * **Root Abuse**: Exploiting standard root tools to install packages or load kernel modules. (Defended by enforcing a non-root User ID (`uid != 0`) and dropping ALL Linux capabilities).
  
* **Network Abuse**:
  * **Internal Reconnaissance**: Scanning the VPC, accessing Redis, or hitting internal APIs. (Defended by setting the network namespace to `none`).
  * **External Abuse**: Mining cryptocurrency, launching DDoS attacks. (Defended by `none` networking).

* **Kernel Attack Surface Exploration**:
  * **Syscall Abuse**: Probing obscure or dangerous syscalls (like `mount`, `unshare`, `ptrace`). (Defended by Seccomp-BPF profiles).

## 3. Out-of-Scope Attacks

* **Zero-Day Linux Kernel Escapes**: If an attacker finds a novel flaw in the Linux kernel namespace or cgroups implementation itself, our Docker-based sandbox could theoretically be breached. Mitigating this would require a stronger hypervisor boundary (e.g. Firecracker, gVisor).
* **Hardware-Level Side Channels (e.g. Spectre/Meltdown)**: Leaking data across CPU caches is out-of-scope for this phase, as tenant data is ephemeral and the host is dedicated.
* **Malicious Worker Binaries**: If the worker process itself is compromised via a supply chain attack.

## 4. Defense In Depth Architecture

Instead of relying solely on the container abstraction, we enforce layers of security:

1. **Non-Root Execution**: Code runs as `uid=1000`, neutering most OS-level modification vectors.
2. **Read-Only Root Filesystem**: The entire container is immutable except for a dedicated, bounded `/tmp` RAM disk.
3. **Dropped Capabilities**: All `CAP_*` permissions are removed.
4. **Namespace Isolation**: Disconnected Network, separate PID tree, separate Mount tree.
5. **Cgroups Limits**: Strict CPU, Memory, and PID ceilings.
6. **Seccomp Filtering**: A restricted subset of safe system calls.
7. **Timeout Hard-Kills**: Execution trees are aggressively SIGKILL'd after a wall-clock limit, ensuring no zombie processes are left behind.
