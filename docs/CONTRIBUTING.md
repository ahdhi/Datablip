# Contributing to DataBlip

First off, thank you for considering contributing to DataBlip\! It's people like you that make open source such a great community. We welcome any type of contribution, from reporting bugs and submitting feature requests to writing code and improving documentation.

This document provides guidelines for contributing to the project.

## Code of Conduct

This project and everyone participating in it is governed by a [Code of Conduct](https://www.google.com/search?q=CODE_OF_CONDUCT.md). By participating, you are expected to uphold this code. Please report unacceptable behavior to the project maintainers.

## How Can I Contribute?

### Reporting Bugs

If you find a bug, please ensure it hasn't already been reported by searching the [Issues](https://www.google.com/search?q=https://github.com/govind1331/Datablip/issues) on GitHub.

If you're unable to find an open issue addressing the problem, [open a new one](https://www.google.com/search?q=https://github.com/govind1331/Datablip/issues/new). Be sure to include a **title and clear description**, as much relevant information as possible, and a **code sample or an executable test case** demonstrating the expected behavior that is not occurring.

### Suggesting Enhancements

If you have an idea for an enhancement, please open an issue to start a discussion. This allows us to coordinate our efforts and prevent duplication of work.

### Your First Code Contribution

Unsure where to begin contributing to DataBlip? You can start by looking through `good first issue` and `help wanted` issues.

Ready to contribute code? Here’s how to set up `datablip` for local development.

1.  **Fork the repository.**
2.  **Clone your fork** locally:
    ```sh
    git clone https://github.com/govind1331/Datablip.git
    ```
3.  **Navigate to the project directory:**
    ```sh
    cd datablip
    ```
4.  **Create a new branch** for your changes:
    ```sh
    git checkout -b feature/your-amazing-feature
    ```
5.  **Make your changes** to the code.
6.  **Run tests and linters** to ensure your changes follow the project's style and don't break existing functionality. The `Makefile` provides commands for this.
    ```sh
    make fmt
    make lint
    make test
    ```
7.  **Commit your changes** with a descriptive commit message. See our [Git Commit Messages](https://www.google.com/search?q=%23git-commit-messages) guide below.
    ```sh
    git commit -m "feat: Add amazing new feature"
    ```
8.  **Push your branch** to your fork on GitHub:
    ```sh
    git push origin feature/your-amazing-feature
    ```
9.  **Open a Pull Request** to the `main` branch of the original `datablip` repository.

## Pull Request Process

1.  Ensure your PR includes a clear description of the problem and solution. Include the relevant issue number if applicable.
2.  The project maintainers will review your Pull Request.
3.  Once your PR is opened, automated CI checks will run against it, as defined in our build workflows. All checks must pass.
4.  We may ask for changes to your code. We aim to collaborate with you to get your contribution merged.

## Styleguides

### Git Commit Messages

We follow the [Conventional Commits](https://www.conventionalcommits.org/en/v1.0.0/) specification. This helps us automate changelogs and makes the project history easier to read. Your commit messages should be structured as follows:

```
<type>[optional scope]: <description>

[optional body]

[optional footer(s)]
```

**Common types:**

  * **feat**: A new feature
  * **fix**: A bug fix
  * **docs**: Documentation only changes
  * **style**: Changes that do not affect the meaning of the code (white-space, formatting, etc)
  * **refactor**: A code change that neither fixes a bug nor adds a feature
  * **test**: Adding missing tests or correcting existing tests
  * **chore**: Changes to the build process or auxiliary tools

### Go Styleguide

  * We use the standard `gofmt` for code formatting. The `make fmt` command will format your code automatically.
  * We use `golangci-lint` for linting. Please run `make lint` before committing to catch any issues.

Thank you for your contribution\!