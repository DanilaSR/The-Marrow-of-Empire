---
name: critical-design-reviews
description: User wants adversarial, critical design reviews — find problems, do not validate; severity-tagged findings with concrete recommendations
metadata:
  type: feedback
---

When asked to review a design document, deliver a critical adversarial review: find what is wrong, underspecified, or missing — do NOT validate the design.

**Why:** the user explicitly framed the AI Dialogue Engine review (2026-08-09) as "Your job is NOT to validate this design — your job is to find narrative and world-consistency problems. Be specific and critical." The project runs formal review gates (MAJOR REVISION status on that GDD), and the user treats reviews as the mechanism for catching problems.

**How to apply:** structure findings as severity-tagged issues (HIGH/MED/LOW) with: the problem, why it matters narratively (cite the specific GDD lines and pillar), and a concrete recommendation. Check pillar alignment explicitly (Every Voice Holds a Piece, World Remembers, Earned Revelation). Reference specific mechanics, not vibes. Credit what works only briefly, if at all.
