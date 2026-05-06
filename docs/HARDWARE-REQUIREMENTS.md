# Hardware Requirement Specification (HRS)

**Project:** MTWS – Multithreaded Secure HTTP Server  
**Type:** Educational Network Security Demo  
**Target Deployment:** Single-machine development, classroom lab, security research  
**Date:** May 2026

---

## 1. Processor (CPU)

### Minimum (Development / Single-client testing)
- **Cores:** 2 cores (or hyperthreading equivalent)
- **Architecture:** x86-64 or ARM64
- **Frequency:** ≥2.0 GHz
- **Example:** Intel Celeron, AMD Ryzen 3, Apple M1 base

### Recommended (Lab / Demo with moderate load)
- **Cores:** 4–8 cores
- **Architecture:** x86-64 or ARM64  
- **Frequency:** ≥2.5 GHz
- **L3 Cache:** ≥4 MB
- **Example:** Intel i5-7th Gen or newer, AMD Ryzen 5, Apple M1/M2

### Justification
- Worker pool (default 10) benefits from 4+ logical cores for true parallelism
- HTTP parsing and WAF scanning are CPU-bound (string matching, automaton traversal)
- Tail latency improves with more cores available for context switching overhead

---

## 2. Memory (RAM)

### Minimum (Development)
- **Total System RAM:** 2 GB
- **Available for MTWS:** ≥256 MB
- **MTWS Process Baseline:** ~20–30 MB (Go runtime + code)

### Recommended (Lab / Demo with 100+ concurrent clients)
- **Total System RAM:** 4–8 GB
- **Available for MTWS:** ≥512 MB
- **Per-connection overhead:** ~4 KB (bufio buffers) + ~1 KB (request/response structures)
- **Estimated peak for 200 concurrent requests:** ~150 MB (connections + worker goroutines + rate limiter state)

### Memory Breakdown (per request)
| Component | Size |
|-----------|------|
| bufio.Reader (~4KB buffer) | 4 KB |
| bufio.Writer (~4KB buffer) | 4 KB |
| HTTP Request struct | ~1 KB |
| Response buffers | ~1 KB |
| WAF automaton state | <1 KB |
| Rate limiter per-IP bucket | ~64 bytes |
| **Total per concurrent conn** | ~10 KB |

### Justification
- Go runtime is memory-efficient; modest overhead for goroutines
- Rate limiter stores one token bucket per unique client IP (sparse for single-lab scenario)
- Handlers may allocate additional memory (application-specific)

---

## 3. Storage (Disk)

### Minimum
- **Binary Size:** ~10 MB (compiled MTWS executable)
- **Configuration Files:** ~10 KB (default WAF policy, config)
- **Logs:** ~1 MB/day (normal operation)
- **Total Required:** 50 MB

### Recommended
- **Binary + Libraries:** 50 MB
- **Configuration + Policies:** 100 KB
- **Log retention (7 days):** 10 MB
- **Benchmark results:** 100 MB
- **Docker images (if containerized):** 500 MB (Go + Alpine base)
- **Total Required:** 700 MB

### Justification
- Go binaries are self-contained (no dynamic linkage in typical setup)
- WAF policies and test payloads are small text files
- Benchmark mode may generate large JSON result files
- Docker deployment adds image size but enables reproducible research

---

## 4. Network Interface

### Minimum
- **Speed:** 100 Mbps (FastEthernet)
- **Type:** Wired Ethernet or WiFi (5 GHz recommended)
- **Latency:** <50 ms typical (local network acceptable)

### Recommended
- **Speed:** 1 Gbps (Gigabit Ethernet)
- **Type:** Wired Ethernet (for benchmarking)
- **Latency:** <10 ms (low-jitter network)

### Justification
- Student project typically runs on lab networks or localhost
- 1 Gbps sufficient for ~10,000 RPS with small payloads
- WAF research may involve comparing with nginx; network IO less critical than CPU

---

## 5. Operating System

### Supported

| OS | Version | Notes |
|----|---------|-------|
| Linux | Kernel 4.4+ (glibc 2.17+) | Primary development target; best performance |
| macOS | 10.15+ (Catalina) | Development convenience; slower syscalls than Linux |
| Windows | 10 (Build 19041+) | Via WSL2 or native Go (not recommended for benchmarks) |

### Justified Minimums

- **File descriptor limit:** Ensure ulimit ≥ 1024 (configurable)
- **Ephemeral port range:** Default OS range ≥20,000 (for connection pool testing)
- **TCP backlog:** Default ≥128 (affects queue under burst load)

### Tuning for Lab Deployment

```bash
# Linux: Increase file descriptors for sustained load testing
ulimit -n 10000

# Linux: Increase ephemeral port range (if load-generating from same machine)
sysctl net.ipv4.ip_local_port_range="1024 65535"

# Linux: Reduce TIME_WAIT to allow more connections in lab
sysctl net.ipv4.tcp_fin_timeout=15
```

---

## 6. Typical Lab Configurations

### Scenario 1: Single Developer Laptop (Development & Unit Testing)

| Component | Specification |
|-----------|---------------|
| CPU | 2–4 cores, ≥2.0 GHz |
| RAM | 2 GB (total), 256 MB available |
| Storage | 50 MB free |
| Network | WiFi 5 GHz or Ethernet |
| OS | macOS, Linux, Windows (WSL2) |
| Load target | 1–10 concurrent requests |

**Use case:** Run unit tests, debug parser, develop handlers.

---

### Scenario 2: Classroom Lab (Multiple Students, Side-by-Side Comparison)

| Component | Specification |
|-----------|---------------|
| CPU | 4–8 cores, ≥2.5 GHz per machine |
| RAM | 4–8 GB (total), 500 MB per MTWS instance |
| Storage | 500 MB per machine (binary, policies, logs) |
| Network | 1 Gbps Ethernet, low-latency LAN |
| OS | Ubuntu 20.04 LTS or later |
| Load target | 50–200 concurrent requests per instance |
| Additional | Docker + Docker Compose for nginx baseline |

**Use case:** Run MTWS ↔ ModSecurity comparison, benchmark research.

---

### Scenario 3: Extended Benchmarking (24h stress test)

| Component | Specification |
|-----------|---------------|
| CPU | 8–16 cores, ≥3.0 GHz |
| RAM | 8–16 GB (total), 1–2 GB available |
| Storage | 1 GB free (for benchmark result JSON logs) |
| Network | 10 Gbps (optional; 1 Gbps sufficient for 50k RPS) |
| OS | Linux (Ubuntu 20.04+ or CentOS 8+) |
| Load target | 1000+ sustained concurrent, 50k+ RPS |

**Use case:** Stability testing, memory leak detection, sustained load analysis.

---

## 7. Dependencies & External Hardware (Optional)

### For ModSecurity Comparison
- **Docker daemon** running (add ~200 MB RAM)
- **nginx container image** (~100 MB disk)
- **ModSecurity CRS ruleset** (~20 MB)
- **Second network interface** (optional, for isolated test network)

### For Network Capture & Analysis (Research)
- **libpcap** + **tcpdump/Wireshark** (~50 MB)
- Adds negligible CPU overhead when passive

### For Container Orchestration (Extended Deployments)
- **Kubernetes node** (if scaling beyond single host)
- Min: 2 GB RAM, 10 GB storage per node
- Not required for student project or single-machine demo

---

## 8. Power & Thermal

### Development (Laptop)
- **Power consumption:** 10–20 W (during benchmarking, depends on CPU/disk)
- **Thermal:** Standard laptop cooling sufficient; monitor CPU temp ~70–80°C

### Lab Server (24h deployment)
- **Power consumption:** 50–150 W (idle ~30 W, peak ~150 W with heavy load)
- **Thermal:** Standard server cooling or rack mount with airflow
- **UPS:** Recommended for unattended benchmarking (protects against power loss)

---

## 9. Constraints & Limitations

| Constraint | Value | Reason |
|-----------|-------|--------|
| Single machine (no cluster) | Max ~50k RPS | OS file descriptor limits, single CPU scheduler queue |
| Worker pool size | Typical 10–100 | Beyond ~100, context switching overhead dominates |
| Concurrent connections | Practical limit ~10,000 | OS tuning + memory |
| Request size | 1 MiB body max | Buffer + parser design limit |
| Request line length | 4096 bytes | Parser constant (HTTP/1.1 ambiguity avoidance) |

---

## 10. Summary Table

| Resource | Minimum | Recommended | Peak (Stress) |
|----------|---------|------------|---------------|
| CPU Cores | 2 | 4–8 | 8–16 |
| RAM | 2 GB | 4–8 GB | 8–16 GB |
| Storage | 50 MB | 500 MB | 1 GB |
| Network | 100 Mbps | 1 Gbps | 1–10 Gbps |
| Typical Load | <10 RPS | 100–1k RPS | 10k–50k RPS |

---

## 11. Validation Checklist

Before deployment, verify:

- [ ] CPU: ≥2 cores available; `cat /proc/cpuinfo | grep processor | wc -l`
- [ ] RAM: ≥500 MB free; `free -h`
- [ ] Storage: ≥100 MB free; `df -h`
- [ ] File descriptors: ≥1000 available; `ulimit -n`
- [ ] Go version: ≥1.26; `go version`
- [ ] Docker (if needed): Running; `docker ps`
- [ ] Network: Latency <50 ms; `ping localhost` or `ping <remote>`
- [ ] Logs: Writable; `touch /var/log/mtws.log`

