export type Lang = 'pt' | 'en';
export type Localized = { pt: string; en: string };
export type Family = 'ROLE' | 'WEAPON' | 'FEAT' | 'TOOL' | 'MECHANISM' | 'STAGE' | 'ARTIFACT';
export interface FeatureSub { kind?: 'ext' | 'opt'; family?: Family; name: Localized; }
export interface FeatureNode { key: string; family: Family; name: Localized; planned?: boolean; sub?: FeatureSub[]; note?: Localized; }
export interface FeatureDefinition { id: string; glyph: string; title: Localized; tag: Localized; how: Localized[]; origin: Array<[Localized, Localized]>; rules: Localized[]; flow: FeatureNode[]; relations: Localized[]; planned?: boolean; }
export const FEATURE_ORDER = [
  "treasure",
  "effort",
  "cartography",
  "initiative",
  "opportunity",
  "crit",
  "roster",
  "dojo"
] as const;
export const FAMILY_LABELS = {
  pt: {
    ROLE: "quem assume",
    WEAPON: "especialidade",
    FEAT: "julgamento",
    TOOL: "operação",
    MECHANISM: "regra garantida",
    STAGE: "fluxo",
    ARTIFACT: "resultado"
  },
  en: {
    ROLE: "who owns it",
    WEAPON: "specialty",
    FEAT: "judgment",
    TOOL: "operation",
    MECHANISM: "enforced rule",
    STAGE: "flow",
    ARTIFACT: "result"
  }
} as const;
export const FEATURE_UI = {
  pt: {
    eyebrow: "o grimório",
    title: "Funcionalidades",
    desc: "Cada funcionalidade reúne os papéis, armas, tools e regras que trabalham juntos. Escolha uma para abrir a ficha completa.",
    legend: "famílias",
    comp: "composição",
    how: "como funciona",
    origin: "origem",
    rules: "regras que garantem",
    planned: "planejado",
    evolving: "em evolução",
    live: "publicado",
    optional: "opcional",
    via: "via",
    fam: {
      ROLE: "quem assume",
      WEAPON: "especialidade",
      FEAT: "julgamento",
      TOOL: "operação",
      MECHANISM: "regra garantida",
      STAGE: "fluxo",
      ARTIFACT: "resultado"
    },
    tbd: "a definir"
  },
  en: {
    eyebrow: "the grimoire",
    title: "Features",
    desc: "Each feature gathers the roles, weapons, tools and rules that work together. Pick one to open its full sheet.",
    legend: "families",
    comp: "composition",
    how: "how it works",
    origin: "origin",
    rules: "rules that guarantee it",
    planned: "planned",
    evolving: "evolving",
    live: "published",
    optional: "optional",
    via: "via",
    fam: {
      ROLE: "who owns it",
      WEAPON: "specialty",
      FEAT: "judgment",
      TOOL: "operation",
      MECHANISM: "enforced rule",
      STAGE: "flow",
      ARTIFACT: "result"
    },
    tbd: "to be defined"
  }
} as const;
export const FEATURES: Record<(typeof FEATURE_ORDER)[number], FeatureDefinition> = {
  treasure: {
    id: "treasure",
    glyph: "❖",
    planned: false,
    title: {
      pt: "Baú do tesouro",
      en: "Treasure chest"
    },
    tag: {
      pt: "Conhecimento offline que vira joia.",
      en: "Offline knowledge that becomes a jewel."
    },
    how: [
      {
        pt: "O Jeweler avalia, lapida e cura conhecimento reutilizável. Para isso empunha a arma Treasure Chest, que organiza fontes offline em Gems e Scrolls.",
        en: "Jeweler evaluates, cuts and curates reusable knowledge. It wields the Treasure Chest weapon, which organizes offline sources into Gems and Scrolls."
      },
      {
        pt: "No uso: index descobre e propõe joias a partir dos documentos completos; mine é a curadoria humana que promove as joias propostas.",
        en: "In practice: index discovers and proposes jewels from the full documents; mine is the human curation step that promotes proposed jewels."
      }
    ],
    origin: [
      [
        {
          pt: "Papel",
          en: "Role"
        },
        {
          pt: "Jeweler · pluggable",
          en: "Jeweler · pluggable"
        }
      ],
      [
        {
          pt: "Arma",
          en: "Weapon"
        },
        {
          pt: "Treasure Chest · importada via skill-for-hire (repositório externo)",
          en: "Treasure Chest · imported via skill-for-hire (external repository)"
        }
      ],
      [
        {
          pt: "Artefatos",
          en: "Artifacts"
        },
        {
          pt: "Gems e Scrolls pertencem ao domínio da Treasure Chest",
          en: "Gems and Scrolls belong to the Treasure Chest domain"
        }
      ]
    ],
    rules: [
      {
        pt: "Estados da joia: proposta → aceita/verificada → depreciada",
        en: "Jewel states: proposed → accepted/verified → deprecated"
      },
      {
        pt: "O documento-fonte completo segue disponível via source card",
        en: "The full source document stays available via a source card"
      }
    ],
    flow: [
      {
        key: "jeweler",
        family: "ROLE",
        name: {
          pt: "Jeweler",
          en: "Jeweler"
        },
        planned: true,
        sub: [],
        note: {
          pt: "Joalheiro · avalia, lapida e cura",
          en: "Jeweler · evaluates, cuts and curates"
        }
      },
      {
        key: "chest",
        family: "WEAPON",
        name: {
          pt: "Treasure Chest",
          en: "Treasure Chest"
        },
        planned: false,
        sub: [
          {
            kind: "ext",
            name: {
              pt: "via skill-for-hire",
              en: "via skill-for-hire"
            }
          }
        ],
        note: {
          pt: "Fontes offline indexáveis e listáveis por comando",
          en: "Offline sources that can be indexed and listed by command"
        }
      },
      {
        key: "gems",
        family: "ARTIFACT",
        name: {
          pt: "Gem / Scroll",
          en: "Gem / Scroll"
        },
        planned: false,
        sub: [],
        note: {
          pt: "Pontos de conhecimento compactos, vinculados à fonte",
          en: "Compact knowledge points linked to their source"
        }
      }
    ],
    relations: [
      {
        pt: "equipa",
        en: "equips"
      },
      {
        pt: "produz",
        en: "produces"
      }
    ]
  },
  effort: {
    id: "effort",
    glyph: "⚖",
    planned: false,
    title: {
      pt: "Esforço proporcional",
      en: "Proportional effort"
    },
    tag: {
      pt: "Sharpshooter · Precise Shot · Leveling",
      en: "Sharpshooter · Precise Shot · Leveling"
    },
    how: [
      {
        pt: "Os três funcionam como uma só engrenagem. O Sharpshooter conduz a avaliação das evidências; a arma Precise Shot mede e normaliza, produzindo um Confidence Report; o Leveling consome esse relatório e resolve esforço, orçamento e condições de escalonamento da missão.",
        en: "The three work as a single gear. Sharpshooter conducts the evidence assessment; the Precise Shot weapon measures and normalizes, producing a Confidence Report; Leveling consumes that report and resolves the mission’s effort, budget and escalation conditions."
      },
      {
        pt: "O JEV é um método opcional por trás da Precise Shot: pode deixar o cálculo mais determinístico, mas o contrato de confiança não depende dele — nem o Leveling.",
        en: "JEV is an optional method behind Precise Shot: it can make the calculation more deterministic, but the confidence contract does not depend on it — and neither does Leveling."
      }
    ],
    origin: [
      [
        {
          pt: "Papel",
          en: "Role"
        },
        {
          pt: "Sharpshooter · pluggable",
          en: "Sharpshooter · pluggable"
        }
      ],
      [
        {
          pt: "Arma",
          en: "Weapon"
        },
        {
          pt: "Precise Shot · externa, fornecida pelo skill-for-hire",
          en: "Precise Shot · external, supplied by skill-for-hire"
        }
      ],
      [
        {
          pt: "Tool",
          en: "Tool"
        },
        {
          pt: "Leveling · fixa e canônica",
          en: "Leveling · fixed and canonical"
        }
      ],
      [
        {
          pt: "Método",
          en: "Method"
        },
        {
          pt: "JEV (opcional) ou método atual, atrás do contrato da arma",
          en: "JEV (optional) or the current method, behind the weapon’s contract"
        }
      ]
    ],
    rules: [
      {
        pt: "Relatório ausente ou inválido nunca vira confiança favorável",
        en: "A missing or invalid report never becomes favorable confidence"
      },
      {
        pt: "Mais confiança não dispensa aprovações nem validações",
        en: "More confidence does not waive approvals or validations"
      },
      {
        pt: "Reavaliações têm limite de tentativas e custo",
        en: "Reassessments have attempt and cost limits"
      },
      {
        pt: "Cada resolução é identificável e preservada",
        en: "Every resolution is identifiable and preserved"
      }
    ],
    flow: [
      {
        key: "sharp",
        family: "ROLE",
        name: {
          pt: "Sharpshooter",
          en: "Sharpshooter"
        },
        planned: true,
        sub: [],
        note: {
          pt: "Conduz a avaliação das evidências",
          en: "Conducts the evidence assessment"
        }
      },
      {
        key: "shot",
        family: "WEAPON",
        name: {
          pt: "Precise Shot",
          en: "Precise Shot"
        },
        planned: true,
        sub: [
          {
            kind: "ext",
            name: {
              pt: "via skill-for-hire",
              en: "via skill-for-hire"
            }
          },
          {
            kind: "opt",
            name: {
              pt: "JEV · opcional",
              en: "JEV · optional"
            }
          }
        ],
        note: {
          pt: "Mede, calcula e normaliza a confiança",
          en: "Measures, computes and normalizes confidence"
        }
      },
      {
        key: "report",
        family: "ARTIFACT",
        name: {
          pt: "Confidence Report",
          en: "Confidence Report"
        },
        planned: false,
        sub: [],
        note: {
          pt: "Valores, método, evidências e limitações",
          en: "Values, method, evidence and limitations"
        }
      },
      {
        key: "valid",
        family: "MECHANISM",
        name: {
          pt: "Confidence validation",
          en: "Confidence validation"
        },
        planned: false,
        sub: [],
        note: {
          pt: "Valida estrutura, aplicabilidade e rastreabilidade",
          en: "Validates structure, applicability and traceability"
        }
      },
      {
        key: "leveling",
        family: "TOOL",
        name: {
          pt: "Leveling",
          en: "Leveling"
        },
        planned: false,
        sub: [],
        note: {
          pt: "Aplica as políticas de proporcionalidade",
          en: "Applies the proportionality policies"
        }
      },
      {
        key: "resolution",
        family: "ARTIFACT",
        name: {
          pt: "Effort Resolution",
          en: "Effort Resolution"
        },
        planned: false,
        sub: [],
        note: {
          pt: "Esforço, orçamento e condições de escalonamento",
          en: "Effort, budget and escalation conditions"
        }
      },
      {
        key: "budget",
        family: "MECHANISM",
        name: {
          pt: "Budget enforcement",
          en: "Budget enforcement"
        },
        planned: false,
        sub: [],
        note: {
          pt: "Garante que a execução respeite a resolução vigente",
          en: "Ensures execution respects the current resolution"
        }
      }
    ],
    relations: [
      {
        pt: "equipa",
        en: "equips"
      },
      {
        pt: "produz",
        en: "produces"
      },
      {
        pt: "passa por",
        en: "goes through"
      },
      {
        pt: "alimenta",
        en: "feeds"
      },
      {
        pt: "resolve",
        en: "resolves"
      },
      {
        pt: "é aplicada por",
        en: "is applied by"
      }
    ]
  },
  cartography: {
    id: "cartography",
    glyph: "⌘",
    planned: false,
    title: {
      pt: "Cartografia",
      en: "Cartography"
    },
    tag: {
      pt: "O mapa estrutural do território.",
      en: "The structural map of the territory."
    },
    how: [
      {
        pt: "O Cartographer interpreta e mantém o conhecimento estrutural do projeto, apoiado pela arma Cartography.",
        en: "Cartographer interprets and maintains the project’s structural knowledge, backed by the Cartography weapon."
      },
      {
        pt: "O Strategist consome os resultados por contrato público; Atlas e Itinerary continuam pertencendo ao domínio de cartografia.",
        en: "Strategist consumes the results through a public contract; Atlas and Itinerary remain owned by the cartography domain."
      }
    ],
    origin: [
      [
        {
          pt: "Papel",
          en: "Role"
        },
        {
          pt: "Cartographer · pluggable (direção arquitetural)",
          en: "Cartographer · pluggable (architectural direction)"
        }
      ],
      [
        {
          pt: "Arma",
          en: "Weapon"
        },
        {
          pt: "Cartography · provider de cartografia",
          en: "Cartography · cartography provider"
        }
      ],
      [
        {
          pt: "Ownership",
          en: "Ownership"
        },
        {
          pt: "Atlas e Itinerary pertencem ao domínio de cartografia",
          en: "Atlas and Itinerary belong to the cartography domain"
        }
      ]
    ],
    rules: [
      {
        pt: "Recortes exportados preservam proveniência rastreável",
        en: "Exported slices keep traceable provenance"
      }
    ],
    flow: [
      {
        key: "carto",
        family: "ROLE",
        name: {
          pt: "Cartographer",
          en: "Cartographer"
        },
        planned: true,
        sub: [],
        note: {
          pt: "Cartógrafo · interpreta o território",
          en: "Cartographer · interprets the territory"
        }
      },
      {
        key: "cartography",
        family: "WEAPON",
        name: {
          pt: "Cartography",
          en: "Cartography"
        },
        planned: true,
        sub: [],
        note: {
          pt: "Pacote de capacidade de cartografia",
          en: "Cartography capability package"
        }
      },
      {
        key: "atlas",
        family: "ARTIFACT",
        name: {
          pt: "Atlas",
          en: "Atlas"
        },
        planned: false,
        sub: [],
        note: {
          pt: "Estrutura persistente de conhecimento",
          en: "Persistent knowledge structure"
        }
      },
      {
        key: "sextant",
        family: "TOOL",
        name: {
          pt: "Sextant",
          en: "Sextant"
        },
        planned: false,
        sub: [],
        note: {
          pt: "Navegação estrutural",
          en: "Structural navigation"
        }
      },
      {
        key: "itinerary",
        family: "ARTIFACT",
        name: {
          pt: "Itinerary",
          en: "Itinerary"
        },
        planned: false,
        sub: [],
        note: {
          pt: "Rota registrada pelo território",
          en: "Route recorded across the territory"
        }
      }
    ],
    relations: [
      {
        pt: "equipa",
        en: "equips"
      },
      {
        pt: "mantém",
        en: "maintains"
      },
      {
        pt: "navegado por",
        en: "navigated by"
      },
      {
        pt: "traça",
        en: "plots"
      }
    ]
  },
  initiative: {
    id: "initiative",
    glyph: "⧗",
    planned: false,
    title: {
      pt: "Iniciativa",
      en: "Initiative"
    },
    tag: {
      pt: "Avalie seu time antes de começar suas missões.",
      en: "Assess your team before starting your missions."
    },
    how: [
      {
        pt: "A Iniciativa aconselha a composição ao entrar em ação: recomenda Tools úteis, armas já disponíveis, evidências e especialistas para o papel e o fluxo atuais.",
        en: "Initiative advises the composition when entering action: it recommends useful Tools, already-available weapons, evidence and specialists for the current role and flow."
      },
      {
        pt: "É consultiva. O papel decide a estratégia dentro da autoridade disponível e o runtime aplica os limites.",
        en: "It is advisory. The role decides the strategy within its authority and the runtime enforces the limits."
      }
    ],
    origin: [
      [
        {
          pt: "Família",
          en: "Family"
        },
        {
          pt: "FEAT · aconselhamento",
          en: "FEAT · advisory"
        }
      ],
      [
        {
          pt: "Entrada",
          en: "Input"
        },
        {
        pt: "papel, fluxo, fase, estado da missão e a resolução do Leveling, quando existir",
        en: "role, flow, phase, mission state and the Leveling resolution, when present"
        }
      ]
    ],
    rules: [
      {
        pt: "Não reescreve a resolução vigente do Leveling",
        en: "Never rewrites the current Leveling resolution"
      },
      {
        pt: "Não aumenta orçamento, faz binding nem autoriza transições",
        en: "Never raises the budget, binds or authorizes transitions"
      }
    ],
    flow: [
      {
        key: "initiative",
        family: "FEAT",
        name: {
          pt: "Initiative",
          en: "Initiative"
        },
        planned: false,
        sub: [
          {
            family: "TOOL",
            name: {
              pt: "Leveling",
              en: "Leveling"
            }
          }
        ],
        note: {
          pt: "Aconselha a composição",
          en: "Advises the composition"
        }
      },
      {
        key: "role",
        family: "ROLE",
        name: {
          pt: "Papel ativo",
          en: "Active role"
        },
        planned: false,
        sub: [],
        note: {
          pt: "Decide a estratégia dentro da própria autoridade",
          en: "Decides the strategy within its own authority"
        }
      },
      {
        key: "budget",
        family: "MECHANISM",
        name: {
          pt: "Budget enforcement",
          en: "Budget enforcement"
        },
        planned: false,
        sub: [],
        note: {
          pt: "Aplica os limites da resolução vigente",
          en: "Enforces the limits of the current resolution"
        }
      }
    ],
    relations: [
      {
        pt: "aconselha",
        en: "advises"
      },
      {
        pt: "atua sob",
        en: "acts under"
      }
    ]
  },
  opportunity: {
    id: "opportunity",
    glyph: "⚔",
    planned: false,
    title: {
      pt: "Ataque de oportunidade",
      en: "Opportunity attack"
    },
    tag: {
      pt: "Problemas não previstos viram side quests.",
      en: "Unforeseen problems become side quests."
    },
    how: [
      {
        pt: "Durante a ação, percebe uma oportunidade lateral e avalia se a missão principal merece uma ADR. Quando cabe, propõe a ADR como side quest.",
        en: "During action it notices a lateral opportunity and evaluates whether the main mission deserves an ADR. When it does, it proposes the ADR as a side quest."
      },
      {
        pt: "A proposta é notificada no Approval Gate, antes de qualquer materialização.",
        en: "The proposal is reported at the Approval Gate, before any materialization."
      }
    ],
    origin: [
      [
        {
          pt: "Família",
          en: "Family"
        },
        {
          pt: "FEAT · proposta delimitada",
          en: "FEAT · bounded proposal"
        }
      ]
    ],
    rules: [
      {
        pt: "Respeita escopo, orçamento e autorização",
        en: "Respects scope, budget and authorization"
      },
      {
        pt: "Nada é materializado sem aprovação humana",
        en: "Nothing is materialized without human approval"
      }
    ],
    flow: [
      {
        key: "opp",
        family: "FEAT",
        name: {
          pt: "Opportunity Attack",
          en: "Opportunity Attack"
        },
        planned: false,
        sub: [],
        note: {
          pt: "Percebe a oportunidade lateral",
          en: "Notices the lateral opportunity"
        }
      },
      {
        key: "adr",
        family: "ARTIFACT",
        name: {
          pt: "ADR · side quest",
          en: "ADR · side quest"
        },
        planned: false,
        sub: [],
        note: {
          pt: "Proposta fora do escopo principal",
          en: "Proposal outside the main scope"
        }
      },
      {
        key: "gate",
        family: "MECHANISM",
        name: {
          pt: "Approval Gate",
          en: "Approval Gate"
        },
        planned: false,
        sub: [],
        note: {
          pt: "Recebe a notificação antes de materializar",
          en: "Receives the notice before materialization"
        }
      }
    ],
    relations: [
      {
        pt: "propõe",
        en: "proposes"
      },
      {
        pt: "notificada no",
        en: "reported at"
      }
    ]
  },
  crit: {
    id: "crit",
    glyph: "↯",
    planned: false,
    title: {
      pt: "Acerto crítico & Riposte",
      en: "Critical Hit & Riposte"
    },
    tag: {
      pt: "Atalhos contextuais para o fluxo curto.",
      en: "Contextual shortcuts into the short flow."
    },
    how: [
      {
        pt: "Ao reconhecer uma situação elegível, as duas Feats solicitam execução pelo fluxo SHORT em vez de abrir uma missão completa — por exemplo, mover análises concluídas, refinadas ou pendentes para as pastas corretas.",
        en: "On recognizing an eligible situation, both Feats request execution through the SHORT flow instead of opening a full mission — for example, moving completed, refined or pending analyses into the correct folders."
      },
      {
        pt: "Ser elegível não é ter autoridade: o runtime ainda exige escopo, binding válido, orçamento e a aprovação humana que o contrato pedir.",
        en: "Being eligible is not having authority: the runtime still requires scope, a valid binding, budget and whatever human approval the contract demands."
      }
    ],
    origin: [
      [
        {
          pt: "Família",
          en: "Family"
        },
        {
          pt: "FEAT · solicita SHORT",
          en: "FEAT · requests SHORT"
        }
      ],
      [
        {
          pt: "Fluxo",
          en: "Flow"
        },
        {
          pt: "SHORT · execução delimitada",
          en: "SHORT · bounded execution"
        }
      ]
    ],
    rules: [
      {
        pt: "A aprovação cobre o conteúdo realmente executado",
        en: "Approval covers the content actually executed"
      },
      {
        pt: "Contexto incompatível: o runtime interrompe ou transiciona para FULL",
        en: "Incompatible context: the runtime interrupts or transitions to FULL"
      }
    ],
    flow: [
      {
        key: "feats",
        family: "FEAT",
        name: {
          pt: "Critical Hit / Riposte",
          en: "Critical Hit / Riposte"
        },
        planned: false,
        sub: [],
        note: {
          pt: "Reconhecem a situação elegível",
          en: "Recognize the eligible situation"
        }
      },
      {
        key: "short",
        family: "STAGE",
        name: {
          pt: "SHORT",
          en: "SHORT"
        },
        planned: false,
        sub: [],
        note: {
          pt: "Execução delimitada com preparação proporcional",
          en: "Bounded execution with proportional preparation"
        }
      },
      {
        key: "approval",
        family: "MECHANISM",
        name: {
          pt: "Approval rules",
          en: "Approval rules"
        },
        planned: false,
        sub: [],
        note: {
          pt: "Verificadas imediatamente antes de executar",
          en: "Re-checked right before execution"
        }
      },
      {
        key: "sniper",
        family: "ROLE",
        name: {
          pt: "Sniper",
          en: "Sniper"
        },
        planned: false,
        sub: [],
        note: {
          pt: "Executa o escopo autorizado",
          en: "Executes the authorized scope"
        }
      },
      {
        key: "completion",
        family: "ARTIFACT",
        name: {
          pt: "Completion Report",
          en: "Completion Report"
        },
        planned: false,
        sub: [],
        note: {
          pt: "Evidência de conclusão",
          en: "Evidence of completion"
        }
      }
    ],
    relations: [
      {
        pt: "solicita",
        en: "requests"
      },
      {
        pt: "verificado por",
        en: "checked by"
      },
      {
        pt: "libera",
        en: "releases"
      },
      {
        pt: "produz",
        en: "produces"
      }
    ]
  },
  roster: {
    id: "roster",
    glyph: "▣",
    planned: false,
    title: {
      pt: "Roster, Rank & Armas",
      en: "Roster, Rank & Weapons"
    },
    tag: {
      pt: "Como uma arma chega a um papel.",
      en: "How a weapon reaches a role."
    },
    how: [
      {
        pt: "O Wizard conduz o ROSTER: descobre as armas instaladas, resolve quais cada papel pode equipar e formaliza a escolha em um binding reproduzível.",
        en: "Wizard drives the ROSTER: it discovers the installed weapons, resolves which ones each role can equip, and formalizes the choice into a reproducible binding."
      },
      {
        pt: "Uma arma RANKED foi homologada pelo Strategist para um contrato e contexto definidos. Isso simplifica a configuração, mas não substitui integridade, compatibilidade nem evidência da missão.",
        en: "A RANKED weapon was approved by Strategist for a defined contract and context. That simplifies setup, but does not replace integrity, compatibility or mission evidence."
      }
    ],
    origin: [
      [
        {
          pt: "Origem da arma",
          en: "Weapon source"
        },
        {
          pt: "internal · embedded · external",
          en: "internal · embedded · external"
        }
      ],
      [
        {
          pt: "Homologação",
          en: "Approval"
        },
        {
          pt: "ranked · unranked (unranked não é incompatível)",
          en: "ranked · unranked (unranked is not incompatible)"
        }
      ],
      [
        {
          pt: "Identidade",
          en: "Identity"
        },
        {
          pt: "weapon_id + versão + digest",
          en: "weapon_id + version + digest"
        }
      ]
    ],
    rules: [
      {
        pt: "A execução resolve o binding formalizado, nunca a versão mais recente",
        en: "Execution resolves the formalized binding, never the latest version"
      },
      {
        pt: "Versão, digest ou contrato mudou? Nova avaliação de homologação",
        en: "Version, digest or contract changed? Re-evaluate the approval"
      }
    ],
    flow: [
      {
        key: "wizard",
        family: "ROLE",
        name: {
          pt: "Wizard",
          en: "Wizard"
        },
        planned: false,
        sub: [],
        note: {
          pt: "Invocador · instala e configura",
          en: "Summoner · installs and configures"
        }
      },
      {
        key: "roster",
        family: "STAGE",
        name: {
          pt: "ROSTER",
          en: "ROSTER"
        },
        planned: false,
        sub: [],
        note: {
          pt: "Fluxo de preparação para binding",
          en: "Preparation flow for binding"
        }
      },
      {
        key: "discovery",
        family: "TOOL",
        name: {
          pt: "Weapon Discovery",
          en: "Weapon Discovery"
        },
        planned: false,
        sub: [],
        note: {
          pt: "Inventaria as armas disponíveis",
          en: "Inventories the available weapons"
        }
      },
      {
        key: "wroster",
        family: "ARTIFACT",
        name: {
          pt: "Weapon Roster",
          en: "Weapon Roster"
        },
        planned: false,
        sub: [],
        note: {
          pt: "Catálogo das identidades encontradas",
          en: "Catalog of the identities found"
        }
      },
      {
        key: "compat",
        family: "TOOL",
        name: {
          pt: "Compatibility Resolver",
          en: "Compatibility Resolver"
        },
        planned: false,
        sub: [],
        note: {
          pt: "Cruza armas e contratos de papel",
          en: "Crosses weapons with role contracts"
        }
      },
      {
        key: "binding",
        family: "ARTIFACT",
        name: {
          pt: "Weapon Binding",
          en: "Weapon Binding"
        },
        planned: false,
        sub: [
          {
            family: "MECHANISM",
            name: {
              pt: "Ranked validation",
              en: "Ranked validation"
            }
          }
        ],
        note: {
          pt: "Vínculo selecionado, formalizado",
          en: "The selected link, formalized"
        }
      }
    ],
    relations: [
      {
        pt: "conduz",
        en: "drives"
      },
      {
        pt: "usa",
        en: "uses"
      },
      {
        pt: "produz",
        en: "produces"
      },
      {
        pt: "alimenta",
        en: "feeds"
      },
      {
        pt: "formaliza",
        en: "formalizes"
      }
    ]
  },
  dojo: {
    id: "dojo",
    glyph: "⛩",
    title: {
      pt: "Dojo",
      en: "Dojo"
    },
    tag: {
      pt: "Pratique antes que seja necessário.",
      en: "Practice before it matters."
    },
    how: [
      {
        pt: "Um campo de treinamento controlado onde os papéis ensaiam workflows, exercitam seus loadouts e aprimoram a tomada de decisão antes de enfrentar missões reais.",
        en: "A controlled training ground where Roles rehearse workflows, exercise their loadouts, and sharpen decision-making before facing real missions."
      }
    ],
    origin: [
      [
        {
          pt: "Origem",
          en: "Origin"
        },
        {
          pt: "Invocação via CLI",
          en: "CLI invocation"
        }
      ]
    ],
    rules: [
      {
        pt: "Cenário autocontido",
        en: "Self-contained scenario"
      }
    ],
    flow: [],
    relations: []
  }
};
