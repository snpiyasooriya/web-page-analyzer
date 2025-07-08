# Web Page Analyzer: Project Guide

## Overview
This guide provides instructions for building, deploying, and testing the application, along with key design notes.

## Local Development
- Prerequisites: Go (1.24+) must be installed.
- Clone & run:
```bash
git clone https://github.com/snpiyasooriya/web-page-analyzer.git
cd web-page-analyzer
make deps
make run
```

## Docker
- Build & run:
```bash
make docker-build
make docker-run
```

## Testing & Quality
- Run tests:
```bash
make test
```
- Run tests with coverage:
```bash
make test-coverage
```
- Run tests with coverage and generate XML report:
```bash
make test-coverage-xml
```
- Lint & vet:
```bash
make lint
make vet
```

## CI/CD
- The application uses GitHub Actions for continuous integration and deployment.
- Pull requests trigger linting, formatting, and testing.
- Merges to main trigger a Docker build and push to Amazon ECR, followed by a deployment to Amazon ECS.


## Key Design Notes
- **Architecture:** A pragmatic layered architecture was chosen to ensure clear separation of concerns (handler, service, analyzer) while remaining idiomatic and easy to navigate.
- **Concurrency:** A concurrent worker pool is used for checking link accessibility. This significantly improves performance on pages with many links and is a core feature of the application's design.
- **Core Libraries:** Standard Go libraries (net/http, golang.org/x/net/html) were used to demonstrate fundamental skills in backend development and avoid unnecessary dependencies.
- **Scope:** The analyzer processes server-rendered HTML and does not execute client-side JavaScript. This aligns with the focus on backend processing for the core task.

## Future Improvements
- **Dynamic Content:** Extend functionality to support JavaScript-rendered pages by integrating a headless browser library.
- **Caching:** Implement a caching layer (e.g., Redis or an in-memory cache) to store results for frequently analyzed URLs, improving response times.


