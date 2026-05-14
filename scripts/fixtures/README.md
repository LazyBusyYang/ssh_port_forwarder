# 本地测试 fixture 素材

`spf-local-test-ed25519-openssh-inline.pem.b64` 为 **UTF-8 文本形式 OpenSSH 私钥 PEM** 再经 Base64 编码的**单行**文件；仓库内不出现 `BEGIN OPENSSH PRIVATE KEY` 明文，以降低 secret 扫描误报。

与 [`docs/LOCAL_DOCKER_TEST_ENV.md`](../../docs/LOCAL_DOCKER_TEST_ENV.md) 中 `PUBLIC_KEY`（`ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIHdOtIbc4G8PIRvJ/4hdsyc+gVftBS+01nNw71Q66z5K …`）配对。

解码示例（使用 `openssl`，在 macOS / Linux 上行为一致）：

```bash
openssl base64 -d -in scripts/fixtures/spf-local-test-ed25519-openssh-inline.pem.b64 -out /tmp/spf-local-test-key.pem
chmod 600 /tmp/spf-local-test-key.pem
```
