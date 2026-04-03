# ADR 001: Hexagonal Architecture

## Status
Accepted

## Context
We need an architecture that supports:
- Multiple VCS backends (GitHub, GitLab, Bitbucket)
- Multiple project discovery strategies
- Comprehensive unit testing without integration test overhead
- Easy addition of new pipeline steps

## Decision
Adopt Hexagonal Architecture (Ports and Adapters) with clear layer boundaries: domain, ports, application, adapters.

## Consequences
- All external concerns are behind interfaces (ports)
- Adapters can be swapped without changing business logic
- Unit tests use mock implementations of ports
- Slightly more files and indirection than a flat structure
- Strong architectural boundaries prevent accidental coupling
