# 📋 [INC-XXX] Incident Title / Standard Operating Procedure

---

## 📌 Metadata
- **Incident ID**: `INC-XXX`
- **Date / Timestamp**: `YYYY-MM-DD HH:MM UTC`
- **Impacted Service(s)**: *(e.g., Coolify, Docker, Tailscale, PostgreSQL)*
- **Severity Level**: *(Critical / High / Medium / Low)*
- **Lead Engineer**: `mostak`
- **Resolution Status**: *(Resolved / Workaround Implemented / Monitoring)*

---

## 1. ⚠️ Problem Statement & Symptoms
*Describe the failure as observed. Include user-facing symptoms, monitoring alerts, or crash loops.*

- **Observed Behavior**:
- **Diagnostic / Log Output**:
```text
[Paste error messages or container crash logs here]
```

---

## 2. 🔍 Root Cause Analysis (RCA)
*Explain why the failure occurred. Identify the underlying technical or environmental cause.*

- **Root Cause**:
- **Trigger Condition**:
- **Why wasn't this caught earlier?**:

---

## 3. 🛠️ Step-by-Step Resolution Procedures
*List the exact sequence of terminal commands used to fix the issue.*

```bash
# Step 1: Isolate the component
sudo docker stop <container-name>

# Step 2: Apply the correction
# [Command here]

# Step 3: Restart and re-bind
sudo systemctl restart <service-name>
```

---

## 4. ✅ Verification & Quality Assurance
*Demonstrate how the fix was validated.*

```bash
# Run verification check
curl -I http://localhost:<port>
```
**Expected Output**:
```text
HTTP/1.1 200 OK
```

---

## 5. 🛡️ Preventative Measures & Action Items
*Actions taken to prevent this incident from recurring in the future.*

- [ ] Automated monitoring / alert rule added.
- [ ] Added to maintenance cron script.
- [ ] Documentation updated in `/docs`.
