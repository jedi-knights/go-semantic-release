# ADR 002: Explicit DI Container

## Status
Accepted

## Context
The application has many components with interdependencies. We need a way to wire them together cleanly at startup without scattering construction logic across the codebase.

## Decision
Use a simple, explicit DI container (`internal/di/Container`) that constructs the full object graph. The container uses lazy initialization (singletons) and returns concrete types through typed accessor methods. No reflection, no magic — just organized constructor calls.

## Consequences
- All wiring is visible in one file
- Easy to understand and debug
- Adding a new adapter means adding one method to the container
- No framework dependency for DI
- Slightly more boilerplate than a DI framework, but much more transparent
