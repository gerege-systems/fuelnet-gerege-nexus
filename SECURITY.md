# Security Policy

## Supported Versions

The following table lists the versions of `open-gerege-mn-erp` currently supported with security updates:

| Version | Supported          |
| ------- | ------------------ |
| 0.1.x   | :white_check_mark: |
| < 0.1.0 | :x:                |

## Reporting a Vulnerability

We take the security of `open-gerege-mn-erp` seriously. If you believe you have discovered a security vulnerability in this project, please report it to us responsibly.

### How to Report

**Please DO NOT report security vulnerabilities through public GitHub issues.**

Instead, report vulnerabilities directly to the security team:
- **Email**: security@gerege.mn
- **Authors**: Gerege Systems Development Team & Gemini AI

### Information to Include

When reporting a vulnerability, please include:
1. Type of issue (e.g., SQL injection, XSS, broken authentication, rate limiter bypass).
2. Full steps to reproduce the vulnerability (including HTTP request samples or code snippets).
3. Potential impact of the vulnerability.
4. Any suggested remediations or patches.

### Response & Disclosure Process

1. **Acknowledgement**: We will acknowledge receipt of your vulnerability report within 24-48 hours.
2. **Investigation**: The engineering team will investigate and verify the issue.
3. **Patching**: If confirmed, a fix will be implemented, tested, and released as a security patch.
4. **Public Disclosure**: A public security advisory will be issued alongside the release notes.

## Security Features Included

- **Multi-Tenancy Isolation**: Strict `tenant_id` database context scoping to prevent cross-tenant data leaks.
- **Tenant-Level App Gating**: Inactive or uninstalled modules reject access with `403 Forbidden`.
- **Security Headers**: Includes `X-Content-Type-Options: nosniff`, `X-Frame-Options: DENY`, `Content-Security-Policy`, and `HSTS`.
- **IP Rate Limiting**: Throttles brute-force login attempts (`golang.org/x/time/rate`).
- **Path Traversal Guards**: Validates all manifest/app slug parameters against regex `^[a-z0-9-]+$`.
- **Password Security**: Uses `bcrypt` password hashing.
