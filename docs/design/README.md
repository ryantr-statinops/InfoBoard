# InfoBoard — Product design

Design docs translate the Product requirements into a user experience
specification. They define what users see, what they can do, and how the
experience behaves across core mode and optional full mode.

Mockups are layout references. Details visible only in an image do not become
requirements unless they are stated in these design documents.

## Reading order

1. [Information architecture](00-information-architecture.md)
2. [Core flows and screens](01-core-flows-and-screens.md)
3. [UI states and responsive behavior](02-ui-states-and-responsive.md)
4. [Mockup map](03-mockup-map.md)

## Milestone boundary

- M1 owns the dashboard shell, add flow, item list, detail, collections,
  notes, and organization status.
- M3 core owns keyword search, retrieval filters, KPI, and analytics fallback.
- M3 optional full mode owns semantic results, related content, and clusters
  only after explicit full-mode setup.
- M4 owns health, backup, rebuild, and maintenance surfaces.

The design layer does not define API, schema, or implementation task details.
Visual component systems and design tokens are out of scope for this phase.
