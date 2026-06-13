## ADDED Requirements

### Requirement: Public docs have distinct roles
README, docs index, tutorials, architecture guides, examples, and website content SHALL each have a defined role in the user journey.

#### Scenario: Contributor adds documentation
- **WHEN** a contributor adds or updates product-facing documentation
- **THEN** they can identify whether the content belongs in README, docs index, tutorial, architecture guide, example, website page, or release note

### Requirement: Primary surfaces are journey-first
Primary public documentation surfaces SHALL lead with the user journey and product promise before module inventories.

#### Scenario: User opens docs index
- **WHEN** a user opens the primary documentation index
- **THEN** the first navigation choices are organized around getting started, extending, deploying, learning architecture, and contributing

### Requirement: Content avoids divergent duplication
The documentation system SHALL prefer canonical guides and cross-links over duplicating instructions that can drift.

#### Scenario: Same capability appears in multiple surfaces
- **WHEN** README, docs, website, or examples mention the same capability
- **THEN** they link to a canonical guide or explicitly share the same source of truth for detailed instructions

### Requirement: Bilingual strategy is explicit
The project SHALL define English-first global surfaces with supported Chinese documentation paths.

#### Scenario: User selects language path
- **WHEN** a user enters the documentation from a primary surface
- **THEN** they can identify the English primary path and the supported Chinese path without mixed-language ambiguity in the first screen
