Plano de Implementação: Testes Unitários com Pydantic
Vou implementar uma camada Python sólida de validação e testes para o Strategist Skill, começando pelos schemas existentes.
1. Estrutura Recomendada (a ser criada)
Bashstrategist/
├── python/                      # ← Novo diretório
│   ├── __init__.py
│   ├── models/                  # Pydantic models
│   │   ├── __init__.py
│   │   ├── intake.py
│   │   ├── progress.py
│   │   ├── config.py
│   │   └── mission.py
│   ├── validators/              # Validadores customizados
│   ├── tests/                   # Testes unitários
│   │   ├── __init__.py
│   │   ├── test_intake.py
│   │   ├── test_progress.py
│   │   └── test_config.py
│   └── utils.py
├── schemas/                     # (existente)
└── pyproject.toml               # ou requirements.txt
2. Instalando Dependências
Adicione um pyproject.toml na raiz do strategist/:
toml[project]
name = "strategist-skill"
version = "0.1.0"
description = "AI Strategist Skill with Pydantic validation"
requires-python = ">=3.11"
dependencies = [
    "pydantic>=2.10",
    "pyyaml>=6.0",
    "pytest>=8.0",
    "pytest-asyncio>=0.25",
]

[tool.pytest.ini_options]
pythonpath = ["python"]
testpaths = ["python/tests"]

3. Modelos Pydantic Principais
python/models/intake.py
Pythonfrom pydantic import BaseModel, Field, field_validator, model_validator
from typing import Dict, List, Optional, Literal
from enum import Enum

class RiskLevel(str, Enum):
    LOW = "low"
    MEDIUM = "medium"
    HIGH = "high"

class DeliveryStrategy(str, Enum):
    INCREMENTAL = "incremental"
    TOTAL = "total"

class ExecutionIntent(str, Enum):
    PLAN_ONLY = "plan_only"
    PLAN_THEN_EXECUTE = "plan_then_execute"

class MissionConstraints(BaseModel):
    delivery_strategy: DeliveryStrategy = DeliveryStrategy.INCREMENTAL
    legacy_compatibility: bool = False
    execution_intent: ExecutionIntent = ExecutionIntent.PLAN_ONLY
    additional: Dict[str, str] = Field(default_factory=dict)

class MissionIntake(BaseModel):
    task_type: str = Field(..., min_length=3)
    description: str = Field(..., min_length=10)
    risk_level: RiskLevel
    constraints: MissionConstraints = Field(default_factory=MissionConstraints)
    user_context: Optional[str] = None
    priority: Optional[int] = Field(default=50, ge=1, le=100)

    @field_validator('task_type')
    @classmethod
    def validate_task_type(cls, v: str) -> str:
        allowed = {"architecture_analysis", "refactor", "feature", "general", "bugfix"}
        if v.lower() not in allowed:
            raise ValueError(f"Task type inválido. Permitidos: {allowed}")
        return v.lower()

    @model_validator(mode='after')
    def validate_high_risk_constraints(self):
        if self.risk_level == RiskLevel.HIGH and self.constraints.execution_intent == ExecutionIntent.PLAN_THEN_EXECUTE:
            raise ValueError("Missões de alto risco não podem ter execution_intent = plan_then_execute sem aprovação explícita")
        return self

python/models/progress.py
Pythonfrom pydantic import BaseModel, Field
from datetime import datetime
from typing import Optional, Dict, Any

class ProgressEvent(BaseModel):
    phase: str = Field(..., pattern="^(preflight|intake|enrichment|scout|engineer|approval|hunter|learning)$")
    status: Literal["started", "completed", "failed", "approved", "rejected"]
    timestamp: datetime = Field(default_factory=datetime.utcnow)
    mission_id: str
    details: Optional[Dict[str, Any]] = None
    source: Optional[str] = None  # ex: "strategist", "scout-provider"

    class Config:
        json_encoders = {
            datetime: lambda v: v.isoformat()
        }

4. Testes Unitários (Exemplos)
python/tests/test_intake.py
Pythonimport pytest
from models.intake import MissionIntake, MissionConstraints, RiskLevel, ExecutionIntent

def test_valid_mission_intake():
    intake = MissionIntake(
        task_type="architecture_analysis",
        description="Analisar a arquitetura do módulo de pagamento",
        risk_level=RiskLevel.MEDIUM,
        constraints=MissionConstraints(
            delivery_strategy="incremental",
            execution_intent="plan_only"
        )
    )
    assert intake.task_type == "architecture_analysis"
    assert intake.risk_level == RiskLevel.MEDIUM

def test_invalid_task_type():
    with pytest.raises(ValueError):
        MissionIntake(
            task_type="invalid_type",
            description="Teste",
            risk_level=RiskLevel.LOW
        )

def test_high_risk_blocked_execution():
    with pytest.raises(ValueError, match="alto risco"):
        MissionIntake(
            task_type="refactor",
            description="Refatoração crítica",
            risk_level=RiskLevel.HIGH,
            constraints=MissionConstraints(
                execution_intent=ExecutionIntent.PLAN_THEN_EXECUTE
            )
        )

def test_default_constraints():
    intake = MissionIntake(
        task_type="general",
        description="Missão simples",
        risk_level=RiskLevel.LOW
    )
    assert intake.constraints.delivery_strategy == "incremental"
    assert intake.constraints.execution_intent == "plan_only"

Próximos Passos Imediatos

Criar a estrutura de pastas python/models/ e python/tests/
Adicionar pyproject.toml
Rodar:Bashcd strategist
pip install -e .
pytest python/tests/ -v


Quer que eu gere agora todos os arquivos completos?
Posso entregar:

Todos os modelos Pydantic (intake, progress, config, skill)
Suite completa de testes
Validador de active.yaml
Fixtures úteis
GitHub Actions para CI

Diga "GERAR TUDO" ou especifique qual parte quer primeiro.tem algum framework em GO que valha a pena ?
OU melhor, algum "agnostico" dado que o projeto so tem .md e .sh em termos de arquivosÓtima pergunta.
Como o projeto Strategist Skill é atualmente quase 100% .md + .sh, adicionar Python (mesmo leve) cria uma dependência que pode não valer a pena. Vamos avaliar as opções de forma honesta.
1. Frameworks em Go – Vale a pena?
Sim, vale muito se você quiser evoluir para um CLI robusto, rápido e single-binary.
Melhores opções em Go (2026):









































Framework / LibTipoRecomendado?ComentárioCobra + ViperCLI completoMelhor escolhaPadrão da indústria (usado por Kubernetes, Docker, Helm, etc.)urfave/cliCLI leveBoa alternativaMais simples que Cobrago-joe / bubbletea (Charm)TUI (interface bonita)Se quiser wizard bonitoExcelente DXgodanticValidação (estilo Pydantic)InteressanteBiblioteca inspirada diretamente no Pydanticozzo-validation ou github.com/go-playground/validatorValidaçãoRecomendadoLeve e muito usado
Vantagens de Go:

Compila para um único binário (strategist) → zero dependências no alvo.
Extremamente rápido.
Excelente para validação forte + CLI.
Fácil distribuir (releases no GitHub).

Desvantagens:

Curva de aprendizado se você não conhece Go.
Mais código boilerplate que Python + Pydantic.


2. Opção Agnóstica (Recomendação Principal agora)
Como o projeto ainda é majoritariamente shell + markdown, sugiro começar com uma abordagem agnóstica e só migrar para Go depois.
Recomendação #1: JSON Schema + ferramentas leves

Defina seus schemas em JSON Schema (padrão universal).
Use ferramentas que rodam em shell:
check-jsonschema (Python, mas roda via pipx ou Docker — leve)
sourcemeta/jsonschema (CLI em Go — excelente!)
yq + jq para validações básicas
yamllint + JSON Schema


Vantagens:

Totalmente agnóstico (não depende de linguagem).
Schemas ficam como arquivos .schema.json.
Pode validar active.yaml, index.yaml, mission_contracts, etc.
Fácil integrar no preflight.sh.

Recomendação #2: Shell + jq/yq (mais simples)
Manter tudo em shell scripts com validações rigorosas usando jq e yq.
Exemplo rápido:
Bash#!/bin/bash
validate_yaml() {
    if ! yq eval '.' "$1" > /dev/null 2>&1; then
        echo "❌ YAML inválido: $1"
        return 1
    fi
    
    # Validação mais forte com JSON Schema via check-jsonschema
    check-jsonschema --schemafile schemas/mission.schema.json "$1"
}

Minha Recomendação Final para Você
Fase 1 (Agora – Recomendado):

Adote JSON Schema para todos os arquivos YAML importantes.
Use check-jsonschema ou sourcemeta/jsonschema (Go) no preflight.
Mantenha a validação em shell scripts.

Fase 2 (Quando crescer):

Crie um CLI em Go + Cobra + Viper + ozzo-validation (ou godantic).
Isso transforma o projeto em uma ferramenta profissional (strategist command).


Quer que eu faça agora?
Escolha uma das opções abaixo:

"JSON Schema" → Eu gero os schemas principais (mission, active, index, progress) + script de validação shell.
"Go CLI" → Eu gero a estrutura inicial com Cobra + validação (estilo Pydantic).
"Híbrido" → JSON Schema agora + plano de migração para Go.

Qual caminho você prefere?