## ADDED Requirements

### Requirement: Memory storage interface
The system SHALL provide a `MemoryStore` interface for storing and retrieving long-term memories.

#### Scenario: Store new memory
- **WHEN** a memory is added to the store
- **THEN** the system SHALL persist it with UUID, content, importance score, and timestamps

#### Scenario: Retrieve memory by ID
- **WHEN** a memory is requested by ID
- **THEN** the system SHALL return the memory and increment its access count

#### Scenario: Delete memory
- **WHEN** a memory is deleted
- **THEN** the system SHALL remove it from persistent storage

### Requirement: Memory importance scoring
The system SHALL support importance scoring for memories with values between 0 and 1.

#### Scenario: Store memory with importance
- **WHEN** a memory is stored with importance score 0.8
- **THEN** the system SHALL persist the importance score

#### Scenario: Default importance score
- **WHEN** a memory is stored without explicit importance
- **THEN** the system SHALL assign default importance of 0.5

### Requirement: Memory decay calculation
The system SHALL calculate memory relevance using exponential decay based on age and access frequency.

#### Scenario: Calculate decay score
- **WHEN** memory relevance is calculated
- **THEN** the system SHALL apply formula: `Score = Importance * e^(-λ * Age) * log(AccessCount + 1)`

#### Scenario: Recent memory has higher score
- **WHEN** comparing two memories with same importance
- **THEN** the more recently accessed memory SHALL have higher relevance score

### Requirement: Memory retrieval by relevance
The system SHALL support retrieving top-k memories by relevance score.

#### Scenario: Retrieve top memories
- **WHEN** user requests top 10 relevant memories
- **THEN** the system SHALL return memories sorted by decay-adjusted score

#### Scenario: Filter by minimum relevance
- **WHEN** user requests memories with minimum relevance 0.3
- **THEN** the system SHALL return only memories with score >= 0.3

### Requirement: Memory search by content
The system SHALL support searching memories by content similarity.

#### Scenario: Search by keyword
- **WHEN** user searches for "project architecture"
- **THEN** the system SHALL return memories containing matching content

#### Scenario: Search with limit
- **WHEN** user searches with limit 5
- **THEN** the system SHALL return at most 5 most relevant matching memories

### Requirement: Memory persistence backend
The system SHALL use SQLite for memory persistence, maintaining the pure Go constraint.

#### Scenario: Initialize memory store
- **WHEN** memory store is initialized
- **THEN** the system SHALL create SQLite table with required schema

#### Scenario: Persist memory
- **WHEN** a memory is stored
- **THEN** the system SHALL write to SQLite database

### Requirement: Memory cleanup
The system SHALL support automatic cleanup of low-relevance memories.

#### Scenario: Cleanup low relevance memories
- **WHEN** cleanup is triggered with threshold 0.1
- **THEN** the system SHALL delete memories with relevance score below threshold

#### Scenario: Protect important memories
- **WHEN** cleanup is triggered
- **THEN** the system SHALL NOT delete memories with importance >= 0.9
