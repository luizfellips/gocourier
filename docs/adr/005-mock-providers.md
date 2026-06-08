# ADR-005: Mock-First Provider Adapters

## Status
Accepted

## Context
Real email, SMS, push, and webhook providers have heterogeneous APIs, rate limits, and failure modes. MVP must validate the platform pipeline before integrating vendors.

## Decision
Define a `ChannelProvider` port and ship mock implementations for all channels at MVP. Real provider adapters are added in V1 behind the same interface.

## Consequences
- End-to-end pipeline can be tested without external dependencies
- Mock providers simulate transient and permanent failures for retry/DLQ testing
- Provider-specific quirks are isolated to adapter packages

## Alternatives Considered
- Integrate real providers at MVP: rejected; slows delivery and couples tests to third parties
- Single generic HTTP provider: insufficient for channel-specific validation
