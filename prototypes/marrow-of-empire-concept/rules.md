// PROTOTYPE - NOT FOR PRODUCTION
// Question: Does "deep but fractured" NPC knowledge create satisfying investigation?
// Date: 2026-08-08

# The Marrow of Empire — Paper Prototype Rules

## What This Prototype Tests

**Hypothesis**: If the player talks to NPCs using a fractured knowledge system (each NPC
knows their domain, no one knows everything), they will feel like an investigator piecing
together a mystery — CONFIRMED if a player follows a thread chain across 3+ NPCs without
prompting and reports feeling curious rather than confused.

**Riskiest assumption**: The knowledge graph structure creates engaging investigation,
not frustration. If players can't form mental connections between NPCs' knowledge fragments,
the core concept fails regardless of LLM quality or visual polish.

## Setting

**Thornwell** — a border town at the edge of the old empire's heartland. Population: ~200.
The imperial ruins begin half a day's walk east. The town lives off trade, foraging, and
salvaged imperial metalwork.

**The mystery**: Aldric, a traveling scholar, was found dead in the old ruins three days ago.
The official report says "creature attack." The investigation was closed in one day.

**The truth** (what the player can discover):
Aldric discovered that the empire's legendary "Divine Light" was actually a technological
power grid called the Ley Network. The empire wasn't destroyed by divine wrath — it collapsed
when the Ley failed. Captain Veren, secretly a member of the Archivists (an order that
suppresses old empire knowledge, believing it too dangerous), killed Aldric using a rare
Nightshade poison to make it look like a creature attack. Aldric hid his final notes in an
old Ley conduit in the ruins — location known only by piecing together fragments from multiple NPCs.

## The NPCs and Their Knowledge Graphs

Each NPC has a knowledge domain. They answer questions within their domain with genuine
knowledge. Outside their domain, they speculate, refer you to someone else, or genuinely
don't know. No NPC knows the full truth.

---

### NPC 1: HALLA — Tavern Keeper

**Domain**: Town gossip, who talks to whom, relationships, travelers passing through,
what happens in her tavern.

**What Halla knows**:
- Aldric was a regular for the three weeks before his death. Quiet, polite, always had a notebook.
- The night before he died, Aldric met someone in the back corner. She didn't see who — the
  tavern was busy — but the meeting was short and tense. Aldric looked shaken afterward.
- Aldric left a satchel behind that night. She kept it in the back room, meaning to return
  it. She hasn't opened it.
- Three days before Aldric died, Captain Veren and Aldric argued loudly in the street outside
  the tavern. She didn't hear the words, but Veren was angry. Unusual — Veren is normally calm.
- The investigation was closed very quickly. Captain Veren said it was "clearly a beast."
  Halla found this strange — beast attacks usually mean a hunt is organized, but none was.

**What Halla DOES NOT know**:
- What was in the satchel (she hasn't looked)
- Who Aldric met with that night
- What Veren and Aldric argued about
- That Aldric was murdered
- Anything about the old empire

**Halla's biases**: She's a gossip but not a liar. She'll share freely but frames
everything as "what she heard" or "what she saw." She trusts Captain Veren (he keeps
the town safe) and won't easily believe ill of him.

---

### NPC 2: GRETA — Blacksmith

**Domain**: Metalwork, tools, weapons, imperial salvage, trade goods, who buys what.

**What Greta knows**:
- Aldric visited her shop five or six times in the weeks before his death. He was
  fascinated by the old imperial metalwork — said the alloy composition was "impossible"
  for the era.
- Aldric brought her a fragment of metal from the ruins — pale silver, impossibly light,
  warm to the touch even when cold. She couldn't identify it. Aldric was very excited.
  He called it "Ley-touched metal."
- Aldric paid her to repair an old imperial lantern. She thought it was decorative, but
  Aldric was very specific about the lens alignment — said it had to be "optically precise."
- Last week, Aldric said he was "close to something big" and that he'd "found a source"
  in the ruins. He was going back there the day before he died.
- Captain Veren came by two days after Aldric's death asking if Greta had any of Aldric's
  belongings. She said no (truthfully — Aldric took the lantern with him). Veren seemed
  relieved.

**What Greta DOES NOT know**:
- What "Ley-touched" means
- Where exactly Aldric went in the ruins
- What Aldric's "big discovery" was
- That the "impossible alloy" is a Ley conduit material
- That Veren is investigating Aldric's death for the wrong reasons

**Greta's biases**: She's practical and observant about her craft. She liked Aldric —
  he actually listened when she explained metalwork. She's suspicious of why Veren asked
  about Aldric's belongings but hasn't connected any dots.

---

### NPC 3: MAREN — Herbalist

**Domain**: Plants, medicine, poisons, poultices, the deep wilds, foraging routes.

**What Maren knows**:
- She examined Aldric's body (she serves as the town's closest thing to a physician).
  The wounds looked like claw marks at first glance, but on closer inspection they were
  too clean — no tearing, no animal saliva. Something was wrong.
- She found residue of Nightshade extract in the wounds. Nightshade is a rare poison
  from a plant that only grows deep in the eastern wilds. It causes paralysis before
  death — the victim can't fight back. Very few people in town even know it exists.
- Captain Veren is one of her regular customers — he buys wound salve for "training
  accidents." He's bought more than usual in the past month. He also bought belladonna
  (a sedative) two weeks ago — said it was for a horse.
- Veren was very insistent that the death was a creature attack. When Maren mentioned
  the wounds looked unusual, he told her she was "overthinking it" and that "the beast
  was clearly a razor-claw." Maren knows razor-claws don't hunt this far south.
- She did not tell Veren about the Nightshade residue. She was afraid of how he'd react.

**What Maren DOES NOT know**:
- That Veren is the killer
- Where Veren would have gotten Nightshade (only she and one other person know where it grows)
- What Aldric was researching
- That Veren is an Archivist

**Maren's biases**: She's cautious and observant. She didn't like how Veren dismissed her
  findings. She's afraid of Veren but trusts her own expertise. She'll share her medical
  findings if the player earns her trust (by demonstrating they're genuinely investigating,
  not just gossiping).

---

### NPC 4: CAPTAIN VEREN — Watch Captain

**Domain**: Town security, patrols, the ruins (officially), town records, "public safety."

**What Veren says (his official story)**:
- Aldric was killed by a wild beast — a razor-claw, judging by the wounds. The investigation
  is closed.
- The ruins are dangerous. He advises all travelers to stay away. "Nothing there worth dying for."
- Aldric was a naive scholar who went where he shouldn't have. A tragedy, nothing more.
- He has no idea what Aldric was researching. "Some academic nonsense."

**What Veren actually knows (revealed only if confronted with evidence)**:
- He killed Aldric. The Nightshade came from a supply the Archivists maintain.
- He is an Archivist — a member of an order that has suppressed knowledge of the old
  empire's technology for generations. They believe the Ley Network's collapse proves
  humanity isn't ready for that power.
- Aldric had found a working Ley conduit and was about to publish his findings. The
  Archivists couldn't allow it.
- He argued with Aldric three days before the death, trying to warn him off. Aldric
  wouldn't stop.
- He doesn't know where Aldric hid his final notes. That keeps him awake at night.

**What Veren DOES NOT know**:
- Where Aldric's notes are hidden
- That Halla has Aldric's satchel
- That Maren found Nightshade residue and didn't report it
- That Orin the Hermit told Aldric about the Ley conduits

**Veren's biases**: He genuinely believes he's protecting people. The Archivists aren't
  evil in their own minds — they've seen what happens when old empire technology is
  unearthed (the Blackfall Incident, 80 years ago). He killed a man and hates himself for
  it. If confronted with overwhelming evidence, he may confess rather than kill again.

---

### NPC 5: ORIN — The Old Hermit

**Domain**: The old empire's true history, the ruins, the Ley Network, the Archivists.

**What Orin knows**:
- The empire wasn't destroyed by gods. It collapsed when the Ley Network — a continent-spanning
  energy grid — failed catastrophically. The "holy light" of legend was technological.
- The ruins east of Thornwell contain a Ley relay station. Most of it is dead, but
  Orin believes some conduits still hum — the system isn't entirely dead, just dormant.
- An order called the Archivists has suppressed this knowledge for centuries. They're not
  a myth — Orin has seen their mark (a closed eye) on documents and on travelers who ask
  too many questions.
- Aldric visited him two days before his death. He asked about "Ley conduits" and "how
  the old ones channeled power." Orin told him about the relay station and where to find
  an access point — the old well in the eastern ruins.
- Aldric said he'd found something that could "change everything." Orin warned him to be
  careful. Aldric said he'd hidden his notes "where only the Ley remembers."
- The Archivists' mark: a closed eye, usually carved small, often mistaken for a knot
  in wood or a flaw in stone. Orin has seen it on Veren's desk — a small seal on a drawer.

**What Orin DOES NOT know**:
- That Aldric was murdered (he's a hermit, news travels slowly)
- That Veren is an Archivist (he suspects but hasn't confirmed)
- What specific evidence Aldric found
- Where Aldric's notes are hidden ("where only the Ley remembers" is all he knows)

**Orin's biases**: He's old, patient, and has spent decades piecing together the truth.
  He speaks in riddles partly from habit and partly from genuine caution. He'll only share
  his deepest knowledge if the player demonstrates they already know something real — he
  tests visitors by seeing if they ask the right questions.

---

## Player Interaction Rules

### How questioning works

The player can ask an NPC about any topic they've encountered. They can also ask
open-ended questions ("What do you know about Aldric?") or ask about another NPC
("What do you think of Captain Veren?"). The NPC answers from their knowledge
domain, with their biases.

### Thread tracking

As the player discovers connections, they record them:

```
Thread: [NPC name] said [fact] → ask [other NPC] about [topic]
Example: Halla said Veren argued with Aldric → ask Veren about the argument
```

### Investigation journal

The player maintains a simple journal tracking:
- **Known facts**: What's been established
- **Open threads**: Questions to pursue, who to ask
- **Contradictions**: Things that don't add up
- **Hunches**: Unconfirmed connections

### Trust mechanic (simplified)

Some NPCs (Maren, Orin, partially Veren) gate their deepest knowledge behind trust.
To earn trust, the player must demonstrate they already know something real —
showing the NPC that they're doing genuine investigation, not idle gossip.

- **Maren** trusts when the player mentions the wounds were unusual (shows they've
  looked beyond the official story)
- **Orin** trusts when the player asks about Ley conduits or the Archivists by name
  (shows they've done research, not just wandering)
- **Veren** only cracks when confronted with 3+ pieces of hard evidence linking him
  to the murder

---

## What We're Measuring

After a playtest session, the tester answers:

1. Did you follow a thread across 3+ NPCs without being told to?
2. At what point did you feel like you were investigating rather than just asking questions?
3. Did you feel confused at any point? If so, where?
4. Did you feel curious about what you'd discover next?
5. Was there a moment where connecting two fragments felt satisfying?

**Success criteria**: The tester follows at least one 3-NPC thread chain unprompted,
reports at least one moment of "feeling like a detective," and does not report
confusion that made them want to stop.
