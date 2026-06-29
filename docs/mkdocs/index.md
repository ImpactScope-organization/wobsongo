# Welcome to Wobsongo

This documentation provides a comprehensive overview of Wobsongo, an open-source misinformation detection tool. It covers the core architecture, the document processing pipeline, and local development practices.

## Project Overview

Wobsongo is built to process, analyze, and detect misinformation using a robust document processing pipeline and modern web technologies. 

- **Backend Pipeline**: Written in Go (utilizing the Echo framework), providing a high-performance API. It integrates with River for robust job queues, handling heavy tasks like PDF parsing and LLM inference.
- **AI Integration**: Utilizes Dockling for structured document extraction, BGE-M3 for vector embeddings, and LLMs for subject-predicate-object knowledge extraction.

## Important Links

- [GitHub Repository](https://github.com/ImpactScope-organization/wobsongo)

## Local Development

### Prerequisites

- [Jetify's Devbox](https://www.jetify.com/docs/devbox/quickstart/) (ensure it's installed on your system).

Devbox uses the Nix package manager under the hood to provide reproducible development environments. 

By running `devbox shell`, it will automatically download and set up the required versions of the tools specified in the `devbox.json` file, such as:

- Go
- Node.js and pnpm
- Make, PostgreSQL, Air, Swag, etc.

> Note that subsequent commands assume you are inside the `devbox shell`.

### Setup

1.  **Clone the repository**:

    ```bash
    git clone git@github.com:ImpactScope-organization/wobsongo.git
    cd wobsongo
    ```

2.  **Activate Devbox environment**:

    ```bash
    devbox shell
    ```

    You should see your shell prompt change, indicating you are now inside the devbox environment:

    ```bash
    (devbox) ➜  wobsongo git:(main) _
    ```

3.  **Install backend dependencies (inside devbox shell)**:

    ```bash
    go mod download
    ```

    Optionally, run the linters:

    ```bash
    make check
    ```

    See `Makefile` for more details.

    You can run the unit tests with:

    ```bash
    make test-unit
    ```

    ...or the entire test suite with:

    ```bash
    make dbtestup test dbtestdown
    ```

### Running the Project

- **Run the backend (with hot-reloading)**:

  ```bash
  make dev
  ```

- You should see the server running:

```
Starting the server at localhost:8000
```