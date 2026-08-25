# Contributing to Mattermost Guest Auto-Isolation Plugin

Thank you for your interest in contributing to the **Mattermost Guest Auto-Isolation Plugin**! We welcome contributions from everyone.

---

## 🛠️ Development Setup

1. **Prerequisites:**
   * Go `1.22+`
   * Docker & Docker Compose (for local Mattermost server testing)
   * `make`

2. **Clone & Build:**
   ```bash
   git clone https://github.com/kazimshah39/mattermost-plugin-guest-isolation.git
   cd mattermost-plugin-guest-isolation
   make test
   make bundle
   ```

---

## 📋 Pull Request Process

1. Fork the repository and create a new branch from `main`.
2. Ensure any new features include relevant unit tests in `./server/`.
3. Run `make test` and ensure all tests pass cleanly.
4. Open a Pull Request with a clear description of the changes and motivation.

---

## 🐞 Reporting Bugs

If you find a bug, please open an issue on GitHub with:
* Clear description of the bug.
* Mattermost server version and operating system.
* Steps to reproduce.
* Expected vs actual behavior.
