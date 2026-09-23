# 📋 [INC-001] MCP Memory Service S3 ONNX Model Download Failure

---

## 📌 Metadata
- **Incident ID**: `INC-001`
- **Date / Timestamp**: `2026-09-22 16:00 UTC`
- **Impacted Service(s)**: `mcp-memory-service` (`00lpnw9fxnqwf6f8lzonywhk-081811531731`)
- **Severity Level**: High (Service startup crash loop)
- **Lead Engineer**: `mostak`
- **Resolution Status**: **Resolved**

---

## 1. ⚠️ Problem Statement & Symptoms
Following the deployment of `doobidoo/mcp-memory-service:latest` via Coolify, the container failed to achieve a healthy state and was stuck in a crash loop. 

### Diagnostic Log Output
```text
HTTP connection timeout: https://chroma-onnx-models.s3.amazonaws.com/all-MiniLM-L6-v2/onnx.tar.gz
Downloading ONNX model weights failed.
Traceback (most recent call last):
  File "/app/src/mcp_memory_service/embeddings/onnx_embeddings.py", line 42, in download_model
    urllib.request.urlretrieve(DOWNLOAD_PATH, target_path)
URLError: <urlopen error [Errno 110] Connection timed out>
```

---

## 2. 🔍 Root Cause Analysis (RCA)
- **Root Cause**: During initial boot, `mcp-memory-service` attempts to dynamically download the `all-MiniLM-L6-v2` ONNX model weights (~80MB archive) from an external Amazon S3 bucket (`chroma-onnx-models.s3.amazonaws.com`).
- **Network Latency / Egress Drop**: Due to transient upstream packet dropping or MTU negotiation delays over the container bridge network, the download exceeded the application's hardcoded timeout threshold, causing the startup process to exit with a non-zero status.

---

## 3. 🛠️ Step-by-Step Resolution Procedures
To decouple the service startup from external internet connectivity and eliminate cold-start download latency, the model was fetched directly on the host machine and injected into the container's local cache.

### Step 1: Download and Extract Model Weights on Host
```bash
# Create local directory and pull the model archive
mkdir -p ~/onnx_model && cd ~/onnx_model
curl -L -o onnx.tar.gz https://chroma-onnx-models.s3.amazonaws.com/all-MiniLM-L6-v2/onnx.tar.gz

# Extract archive
tar xzf onnx.tar.gz
```

### Step 2: Inject Model Weights into the Container Cache
```bash
# Ensure target cache directory exists inside container
sudo docker exec 00lpnw9fxnqwf6f8lzonywhk-081811531731 \
  mkdir -p /root/.cache/mcp_memory/onnx_models/all-MiniLM-L6-v2

# Copy extracted model files into the container
sudo docker cp ~/onnx_model/onnx 00lpnw9fxnqwf6f8lzonywhk-081811531731:/root/.cache/mcp_memory/onnx_models/all-MiniLM-L6-v2/
```

### Step 3: Restart Container and Inspect Logs
```bash
sudo docker restart 00lpnw9fxnqwf6f8lzonywhk-081811531731
sudo docker logs --tail 25 00lpnw9fxnqwf6f8lzonywhk-081811531731
```

---

## 4. ✅ Verification & Quality Assurance
Querying the MCP service health endpoint returns `HTTP 200 OK`:

```bash
curl -s -o /dev/null -w "%{http_code}\n" http://localhost:8001
# Output: 200
```

The container log confirms:
```text
ONNX model 'all-MiniLM-L6-v2' already available in cache. Skipping download.
Service initialized on port 8000.
```

---

## 5. 🛡️ Preventative Measures
- Stored the extracted model weights in `/home/mostak/onnx_model/` on the host as a persistent backup.
- If the container is ever recreated, mount the host directory directly via Docker Compose volume to prevent future container-level rebuild loss.
