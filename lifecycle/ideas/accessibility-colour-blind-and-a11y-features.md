---
title: 'Accessibility Features Investigation: Colour Blind Palette Support and Beyond'
type: idea
status: draft
lineage: accessibility-colour-blind-and-a11y-features
created: "2026-05-10T10:05:02+10:00"
priority: normal
labels:
    - feature
    - frontend
    - usability
    - enhancement
---

# Accessibility Features Investigation: Colour Blind Palette Support and Beyond

Investigate and implement accessibility improvements across the Innovation Maker UI, with a primary focus on colour blind palette support. This includes auditing the current colour usage in the graph visualisations, status indicators, and general UI components to identify where colour alone is used to convey meaning, and replacing or augmenting those with colour blind-friendly alternatives (e.g. deuteranopia, protanopia, tritanopia-safe palettes).

Beyond colour blindness, the investigation should assess what other accessibility features can be realistically supported: keyboard navigation, screen reader compatibility (ARIA labels, semantic HTML), sufficient contrast ratios (WCAG AA/AAA), focus indicators, and reduced-motion preferences. Each area should be evaluated for implementation cost and user impact.

The output should be a prioritised list of accessibility improvements with recommendations for which to address in v1 versus later milestones, along with any tooling (e.g. axe-core, Lighthouse) that can be integrated into the QA or CI pipeline to prevent regressions.
