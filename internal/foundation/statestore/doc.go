// Package statestore owns Synora's authoritative typed local state store,
// journal, replay, compaction, resident metadata and action outbox persistence.
// Domain records are supplied as compile-time generic types so Foundation
// does not depend on their owning runtime package.
package statestore
