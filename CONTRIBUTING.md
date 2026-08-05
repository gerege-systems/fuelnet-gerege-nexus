# Contributing to open-gerege-mn-erp

Thank you for your interest in contributing to **open-gerege-mn-erp**! We welcome contributions from the community to help build a modular, high-performance open-source ERP platform.

---

## 👥 Authors & Maintainers

This project is developed and maintained by:
- **Gerege Systems Development Team**
- **Gemini AI**

---

## 📜 Code of Conduct

All contributors are expected to adhere to our [Code of Conduct](CODE_OF_CONDUCT.md). Please report unacceptable behavior to `community@gerege.mn`.

---

## 🛠️ How to Contribute

### 1. Reporting Bugs
Before creating a bug report, please check existing issues to ensure it hasn't already been reported.

When opening a bug report:
- Use the **Bug Report Template**.
- Provide clear steps to reproduce the issue.
- Mention your environment (Go version, Node.js version, OS, PostgreSQL version).

### 2. Suggesting Enhancements
Feature requests are welcome! Describe the use case, motivation, and proposed solution clearly.

### 3. Submitting Pull Requests (PRs)
1. **Fork the Repository**: Create your feature branch (`git checkout -b feature/amazing-feature`).
2. **Follow Code Conventions**:
   - Backend: Go 1.24+ standard formatting (`gofmt`), structured logging (`slog`), explicit error handling.
   - Frontend: Next.js 15 App Router, TypeScript strict mode, Tailwind CSS.
3. **Write Unit Tests**: Ensure all new backend code includes unit tests (`*_test.go`).
4. **Run Verification Commands**:
   ```bash
   # Backend tests & build
   cd backend && go test ./... && go build ./...

   # Frontend build
   cd frontend && npm run build
   ```
5. **Commit Messages**: Follow Conventional Commits format:
   - `feat: add invoice management module`
   - `fix: resolve stock level calculation rounding`
   - `docs: update module authoring guide`
6. **Open PR**: Submit your PR targeting the `main` branch.

---

## 🧩 Adding New Business App Modules

To author a new module:
1. Create a package under `backend/internal/apps/<module_name>/`.
2. Implement the `internal.Module` interface defined in `backend/internal/module.go`.
3. Register the module in `appregistry` and create the corresponding JSON manifest under `catalog/manifests/`.
4. Add frontend views under `frontend/app/<module_name>/page.tsx`.

For detailed instructions, read the [Module Authoring Guide](docs/MODULE_AUTHORING_GUIDE.md).
