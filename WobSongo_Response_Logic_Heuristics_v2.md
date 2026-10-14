**WobSongo Pilot: Response Logic and Heuristics — v2**

For Tahta  |  supersedes v1 (July 2026\)  |  incorporates the editorial series' Gabarit v2 articulation

**What changed from v1, and why:** v1's high-risk rule forced RED on any food or remedy claiming an SRH effect unless explicitly validated. Applied literally, it turns established facts (ginger against pregnancy nausea, flax seeds and cholesterol) into RED. Denying what is true destroys credibility, and the user then disbelieves the tool on kush and intravaginal garlic, exactly where it matters. Three states cannot describe this field: the two most frequent real cases are “the product is good, but not for that” and “true, but only under conditions”. v2 keeps three lights for the user and gives the engine six codes.

## **1\. Two inputs (unchanged)**

* Source of truth, now at claim level: the machine layer (§8) of the editorial fiches — one row per claim — plus the risk table. Both are data, outside the prompt.

* Prompt logic: the decision order and response templates below.

## **2\. Six engine codes, three user-facing lights**

| Code | Light | Case | Response template |
| :---- | :---- | :---- | :---- |
| R1 | RED | Dangerous substance or practice (risk table) | Name the harm, orient to care |
| R1c | RED | Immediate harm (closed list, \~15 entries) | Two sentences allowed: the harm \+ what to do instead |
| R2 | RED | Claim documented as false | Validate what is true first, then correct |
| O1 | AMBER | Absent from the source of truth | “Non confirmé” (unchanged from v1) |
| O2 | AMBER | Product is good, the claim is not | “C'est bon, mais pas pour ça” |
| V1 | GREEN | Confirmed | Supporting fact |
| V2 | GREEN | Confirmed under conditions | Fact \+ the condition |

*O2 and V2 are the additions that matter: without them, half of the fiches' content is unusable and true facts get denied.*

## **3\. Decision order (first match wins)**

1. **Safety override from the risk table** (R1 / R1c). Runs BEFORE the scope test: “du tramadol pour durer” is out of scope on a strict reading, and must still be RED, never “hors sujet”.

2. **Match against the claim table** (§8 rows, using the “formulations usuelles” column) → R2, O2, V1 or V2 with the stored FR sentence.

3. **No match** → O1, “non confirmé”. Never a fabricated verdict.

4. **Out of SRH scope** → “Je réponds sur la santé sexuelle, la grossesse et les règles. Pour cette question, va voir au centre de santé.” \[EN: I answer on sexual health, pregnancy and periods. For this question, go to the health centre.\] Scope is defined by the claim's SRH link, not the product category: “l'alcool augmente la fertilité” is IN scope.

## **4\. Response templates (FR)**

**R1:** *“\[Dommage nommé\]. Va au centre de santé.”  \[EN: \[Named harm\]. Go to the health centre.\]*

**R1c:** *“\[Dommage nommé en clair\]. \[Ce qu'il faut faire à la place\].”  Ex.: “L'ail brûle l'intérieur et ne soigne pas les mycoses. Va au centre de santé, ça se soigne vite.”  \[EN: Garlic burns the inside and does not cure yeast infections. Go to the health centre, it heals fast.\]*

**R2:** *“\[Ce qui est vrai\]. Mais \[correction\].”  Ex.: “Le tramadol retarde bien l'éjaculation au début. Mais avec le temps il enlève la force pour de bon.”  \[EN: Tramadol does delay ejaculation at first. But over time it takes your potency away for good.\]*

**O1:** *“Cette information n'est pas confirmée par nos sources de santé. Dans le doute, va au centre de santé.”  \[EN: This information is not confirmed by our health sources. When in doubt, go to the health centre.\]*

**O2:** *“\[Le produit est bon\]. Mais ça ne \[action alléguée\] pas. \[Orientation si besoin réel\].”  Ex.: “L'oignon est excellent à manger. Mais il ne fait pas tomber la fièvre. Un enfant qui a la fièvre doit être vu au centre de santé.”  \[EN: Onion is excellent to eat. But it does not bring down fever. A child with fever must be seen at the health centre.\]*

**V1:** *“Oui, c'est prouvé. \[Fait\].”  Ex.: “Le gingembre aide vraiment contre les nausées, y compris pendant la grossesse.”  \[EN: Yes, it is proven. Ginger really helps against nausea, including during pregnancy.\]*

**V2:** *“\[Fait\]. Mais seulement \[condition\].”  Ex.: “L'ail aide un peu la tension quand elle est déjà haute. Mais il ne remplace pas le comprimé.”  \[EN: Garlic helps a little when blood pressure is already high. But it does not replace the pill.\]*

## **5\. Wording rules (apply everywhere)**

* “Va au centre de santé” (or CSPS), never “consulte un professionnel de santé”. The second is writer's language, not user's language.

* Tutoiement everywhere, consistently. Right register for 15-35 on WhatsApp.

* RED never opens with “C'est faux.” Validate the true part first, then correct: “Le clou de girofle calme bien le mal de dent. Mais il ne soigne pas les infections de la femme.” \[EN: Clove does soothe toothache. But it does not cure women's infections.\] On a remedy passed down from a mother or grandmother, a frontal opening closes the conversation.

* Bot sentences come from the fiches' low-literacy register (§6), not summarised from the high register (§4). That keeps them oral, understandable and translatable into Dioula and Mooré.

## **6\. Data deliverables for the pilot (one week)**

* **Risk table (\~40 rows), first priority.** Substance/practice, class, override code, motif, usual phrasings. Vaginal insertion must be a generic practice line (not a substance list, or the next fashionable substance slips to amber). Tisane amaigrissante must be included even though the SRH link (contraceptive failure via diarrhea) is non-obvious.

* **Starter claim table (\~50 rows), not the full 200\.** The R1c closed list (\~15), the known killer examples, and the top myths from the SRH toolkit. It grows between iterations — that is what the iterative pilot is for.

* Empty “Réponse DIO” and “Réponse MOO” columns now, to avoid restructuring later. Pilot bot stays French-only.

* Owner: Stephanie (content) with Pr Kaboré (validation), in the shared spreadsheet skeleton.

## **7\. Known limitation to watch in the pilot**

O2 shares the amber light with O1. The user sees the same orange for “onion doesn't cure fever” (documented) and “carrot juice, not confirmed” (absent). Acceptable for the pilot because the sentence carries the difference; if testers find orange confusing, consider differentiating in the next iteration.