Proposta de Refatoração de Pacotes – World-Class Engineering
Aqui está uma proposta completa e moderna de estrutura de pacotes para o strategist-skill, seguindo padrões usados em projetos Go de alto nível (Kubernetes, Terraform, Temporal, Istio, etc.).
Estrutura Atual (Resumo)

cmd/strategist/ → CLI handlers
internal/ → compile, domain, embed, install, stale
strategist/ → templates, contracts, personas, schemas, skills (assets + config)

Essa mistura atual é funcional, mas mistura domínio, infraestrutura, assets e aplicação, o que dificulta escalabilidade e testes.

Nova Estrutura Proposta (Clean/Hexagonal Architecture)
Bashstrategist-skill/
├── cmd/
│   └── strategist/
│       └── main.go                  # Entry point mínimo
├── internal/
│   ├── application/                 # Orquestração e use cases
│   │   ├── mission/
│   │   ├── approval/
│   │   ├── guardrails/
│   │   └── orchestrator.go
│   │
│   ├── domain/                      # Core do negócio (mais importante)
│   │   ├── mission/
│   │   │   ├── mission.go
│   │   │   ├── phase.go
│   │   │   ├── status.go
│   │   │   └── aggregate.go
│   │   ├── approval/
│   │   ├── knowledge/
│   │   ├── persona/
│   │   └── errors.go
│   │
│   ├── ports/                       # Interfaces (Hexagonal)
│   │   ├── llm.go
│   │   ├── repository.go
│   │   ├── git.go
│   │   └── filesystem.go
│   │
│   ├── infrastructure/              # Implementações concretas
│   │   ├── llm/
│   │   │   └── claude.go, openai.go, grok.go
│   │   ├── persistence/
│   │   │   └── jsonl.go, git.go
│   │   ├── git/
│   │   └── sandbox/
│   │
│   ├── presentation/                # CLI (cobra commands)
│   │   ├── cmd/
│   │   │   ├── root.go
│   │   │   ├── mission.go
│   │   │   ├── compile.go
│   │   │   └── install.go
│   │   └── cli.go
│   │
│   └── config/                      # Configuração + validação
│       └── config.go
│
├── pkg/                             # Código público (se quiser expor API)
│   └── strategist/
│       └── ...
│
├── strategist/                      # Assets imutáveis (mantido)
│   ├── contracts/
│   ├── personas/
│   ├── roles/
│   ├── schemas/
│   ├── skills/
│   ├── templates/
│   └── skill.yaml
│
├── docs/
├── tests/                           # BDD / integration tests
├── .github/
├── go.mod
└── Makefile

Justificativa dos Principais Pacotes








































PacoteResponsabilidadePrincípio Aplicadointernal/domainRegras de negócio puras, entidades, value objectsDDD + Rich Domain Modelinternal/applicationUse cases / Orquestração (o "cérebro")Application Servicesinternal/portsInterfaces de saída (dependências invertidas)Dependency Inversioninternal/infrastructureAdaptores concretos (LLM, Git, FS, etc)Hexagonal / Ports & Adaptersinternal/presentationCLI, HTTP, etc.Presentation Layerstrategist/Assets estáticos (templates, personas, schemas)Separation of Concerns

Como Migrar (Passo a Passo Recomendado)
Fase 1 (1-2 dias) – Base

Criar a nova estrutura de pastas
Mover internal/domain para o novo local (já existe, só ajustar)
Criar pastas ports, application, infrastructure

Fase 2 (3-4 dias) – Refatoração

Extrair interfaces para ports/
Injetar dependências via google/wire ou uber/fx
Mover lógica de CLI para presentation/cmd/
Refatorar internal/compile, install, stale para infrastructure/ ou application/

Fase 3 – Melhoria

Adicionar wire.go para DI
Criar testes de unidade com mocks das ports


Exemplo de Código (Domain)
Go// internal/domain/mission/mission.go
type Mission struct {
    ID          uuid.UUID
    Title       string
    Spec        Spec
    Phases      []Phase
    Status      Status
    Approvals   []Approval
    Metadata    Metadata
}

func (m *Mission) Validate() error { ... }

func (m *Mission) Approve(gate ApprovalGate) error { ... }

Aqui está a versão completa da estrutura proposta, com todos os arquivos principais já criados como skeletons (arquivos vazios + comentários claros).
1. Nova Estrutura de Diretórios
Bashmkdir -p \
  internal/application/{mission,approval,guardrails} \
  internal/domain/{mission,approval,knowledge,persona} \
  internal/ports \
  internal/infrastructure/{llm,persistence,git,sandbox} \
  internal/presentation/cmd \
  internal/config

2. Arquivos Completos (Skeletons)
internal/domain/mission/mission.go
Gopackage mission

import (
    "github.com/google/uuid"
    "time"
)

// Mission é o Aggregate Root principal
type Mission struct {
    ID          uuid.UUID
    Title       string
    Description string
    Spec        Spec
    Phases      []Phase
    Status      Status
    Approvals   []ApprovalRecord
    Metadata    Metadata
    CreatedAt   time.Time
    UpdatedAt   time.Time
}

// Spec contém a especificação da missão (SDD)
type Spec struct {
    Objective    string
    Constraints  []string
    SuccessCriteria []string
    Persona      string
}

// Phase representa uma etapa da missão
type Phase struct {
    Name        string
    Description string
    Status      PhaseStatus
    Output      string
    ExecutedAt  *time.Time
}

type Status string
type PhaseStatus string

const (
    StatusPending   Status = "pending"
    StatusApproved  Status = "approved"
    StatusInProgress Status = "in_progress"
    StatusCompleted Status = "completed"
    StatusRejected  Status = "rejected"
)

func NewMission(title, description string, spec Spec) (*Mission, error) {
    if title == "" {
        return nil, ErrInvalidTitle
    }
    return &Mission{
        ID:          uuid.New(),
        Title:       title,
        Description: description,
        Spec:        spec,
        Status:      StatusPending,
        CreatedAt:   time.Now(),
        UpdatedAt:   time.Now(),
    }, nil
}

func (m *Mission) Validate() error {
    // invariants
    return nil
}

func (m *Mission) Approve(gate ApprovalGate) error {
    // lógica de aprovação
    return nil
}
internal/domain/errors.go
Gopackage domain

import "errors"

var (
    ErrInvalidTitle      = errors.New("invalid mission title")
    ErrApprovalRequired  = errors.New("approval is required before execution")
    ErrInvalidPersona    = errors.New("invalid persona")
    ErrForbiddenAction   = errors.New("action forbidden by guardrails")
)
internal/ports/ports.go
Gopackage ports

import (
    "context"
    "github.com/SergioLacerda/strategist-skill/internal/domain/mission"
)

// LLM Port
type LLM interface {
    Execute(ctx context.Context, prompt string, persona string) (string, error)
    ExecuteWithSchema(ctx context.Context, prompt string, schema any) (any, error)
}

// Repository Port
type MissionRepository interface {
    Save(ctx context.Context, m *mission.Mission) error
    GetByID(ctx context.Context, id string) (*mission.Mission, error)
    List(ctx context.Context) ([]*mission.Mission, error)
}

// Git Port
type Git interface {
    Commit(ctx context.Context, message string, files []string) error
    Push(ctx context.Context) error
    Status(ctx context.Context) (string, error)
}

// Filesystem Port
type Filesystem interface {
    ReadFile(path string) ([]byte, error)
    WriteFile(path string, data []byte) error
    ListDir(path string) ([]string, error)
}
internal/application/orchestrator.go
Gopackage application

import (
    "context"
    "github.com/SergioLacerda/strategist-skill/internal/domain/mission"
    "github.com/SergioLacerda/strategist-skill/internal/ports"
)

type Orchestrator struct {
    llm          ports.LLM
    repo         ports.MissionRepository
    git          ports.Git
    fs           ports.Filesystem
    guardrails   GuardrailsService
}

func NewOrchestrator(llm ports.LLM, repo ports.MissionRepository, git ports.Git, fs ports.Filesystem) *Orchestrator {
    return &Orchestrator{
        llm:        llm,
        repo:       repo,
        git:        git,
        fs:         fs,
        guardrails: NewGuardrailsService(),
    }
}

func (o *Orchestrator) ExecuteMission(ctx context.Context, spec mission.Spec) (*mission.Mission, error) {
    // 1. Criar missão
    // 2. Validar guardrails
    // 3. Solicitar aprovação
    // 4. Executar fases (Ranger → Archivist → Sniper)
    // 5. Learning loop
    return nil, nil
}
internal/config/config.go
Gopackage config

import "github.com/spf13/viper"

type Config struct {
    LLMProvider string `mapstructure:"llm_provider" validate:"required,oneof=claude openai grok"`
    Model       string `mapstructure:"model"`
    Temperature float64 `mapstructure:"temperature"`
    MaxTokens   int    `mapstructure:"max_tokens"`
    ApprovalMode string `mapstructure:"approval_mode" validate:"oneof=auto manual always"`
}

func Load() (*Config, error) {
    viper.SetConfigName("config")
    viper.SetConfigType("yaml")
    viper.AddConfigPath(".")
    // ...
    return nil, nil
}
internal/presentation/cmd/root.go
Gopackage cmd

import (
    "github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
    Use:   "strategist",
    Short: "Strategist Skill - Cognitive Software Engineering Agent",
    Long:  `A production-grade AI software engineering orchestrator with human approval gates.`,
}

func Execute() {
    rootCmd.Execute()
}

func init() {
    // registrar subcomandos
}
internal/infrastructure/llm/claude.go
Gopackage llm

import "github.com/SergioLacerda/strategist-skill/internal/ports"

type ClaudeClient struct {
    apiKey string
    model  string
}

func NewClaudeClient(apiKey, model string) ports.LLM {
    return &ClaudeClient{apiKey: apiKey, model: model}
}

// Implementar métodos da interface ports.LLM

3. Exemplo Prático de Dependency Injection com google/wire
Crie o arquivo: internal/wire.go
Go//go:build wireinject
// +build wireinject

package internal

import (
    "github.com/google/wire"
    "github.com/SergioLacerda/strategist-skill/internal/application"
    "github.com/SergioLacerda/strategist-skill/internal/config"
    "github.com/SergioLacerda/strategist-skill/internal/infrastructure/llm"
    "github.com/SergioLacerda/strategist-skill/internal/infrastructure/persistence"
    "github.com/SergioLacerda/strategist-skill/internal/ports"
)

func InitializeOrchestrator(cfg *config.Config) (*application.Orchestrator, error) {
    wire.Build(
        provideLLM,
        persistence.NewMissionRepository,
        persistence.NewGitAdapter,
        persistence.NewFilesystemAdapter,
        application.NewOrchestrator,
    )
    return nil, nil
}

func provideLLM(cfg *config.Config) ports.LLM {
    switch cfg.LLMProvider {
    case "claude":
        return llm.NewClaudeClient("...", cfg.Model)
    case "grok":
        return llm.NewGrokClient("...")
    default:
        return llm.NewClaudeClient("...", cfg.Model)
    }
}
Como gerar o código:
Bashgo install github.com/google/wire/cmd/wire@latest
wire ./internal
Isso gera automaticamente o arquivo wire_gen.go.

1. Arquivos Restantes (Guardrails, Approval, Infrastructure, etc.)
internal/domain/approval/approval.go
Gopackage approval

import (
    "time"
    "github.com/google/uuid"
)

type ApprovalRecord struct {
    ID        uuid.UUID
    MissionID uuid.UUID
    Approved  bool
    Reviewer  string
    Comments  string
    Timestamp time.Time
}

type ApprovalGate interface {
    RequestApproval(missionID uuid.UUID, summary string) (ApprovalRecord, error)
    Validate(record ApprovalRecord) error
}
internal/application/guardrails/guardrails.go
Gopackage guardrails

import (
    "context"
    "github.com/SergioLacerda/strategist-skill/internal/domain"
)

type GuardrailsService struct {
    forbiddenActions []string
    maxPhases        int
}

func NewGuardrailsService() *GuardrailsService {
    return &GuardrailsService{
        forbiddenActions: []string{"rm -rf", "sudo", "format", "drop database"},
        maxPhases:        20,
    }
}

func (g *GuardrailsService) ValidateMission(ctx context.Context, spec domain.Spec) error {
    for _, constraint := range spec.Constraints {
        for _, forbidden := range g.forbiddenActions {
            if containsIgnoreCase(constraint, forbidden) {
                return domain.ErrForbiddenAction
            }
        }
    }
    return nil
}

func containsIgnoreCase(s, substr string) bool { /* implementação */ return false }
internal/application/mission/service.go
Gopackage mission

import (
    "context"
    "github.com/SergioLacerda/strategist-skill/internal/domain/mission"
    "github.com/SergioLacerda/strategist-skill/internal/ports"
)

type Service struct {
    orchestrator *application.Orchestrator // se precisar de referência circular, ajustar
}

func (s *Service) CreateAndExecute(ctx context.Context, title, desc string, spec mission.Spec) (*mission.Mission, error) {
    m, err := mission.NewMission(title, desc, spec)
    if err != nil {
        return nil, err
    }
    // ... lógica completa
    return m, nil
}
internal/infrastructure/persistence/mission_repo.go
Gopackage persistence

import (
    "context"
    "github.com/SergioLacerda/strategist-skill/internal/domain/mission"
    "github.com/SergioLacerda/strategist-skill/internal/ports"
)

type MissionRepository struct {
    fs ports.Filesystem
}

func NewMissionRepository(fs ports.Filesystem) ports.MissionRepository {
    return &MissionRepository{fs: fs}
}

func (r *MissionRepository) Save(ctx context.Context, m *mission.Mission) error {
    // salvar como JSON + append em .missions.jsonl
    return nil
}

func (r *MissionRepository) GetByID(ctx context.Context, id string) (*mission.Mission, error) {
    return nil, nil
}
internal/infrastructure/git/git.go
Gopackage git

import (
    "context"
    "github.com/SergioLacerda/strategist-skill/internal/ports"
)

type Adapter struct{}

func NewGitAdapter() ports.Git {
    return &Adapter{}
}

func (a *Adapter) Commit(ctx context.Context, message string, files []string) error { return nil }
func (a *Adapter) Push(ctx context.Context) error { return nil }
internal/infrastructure/filesystem/fs.go
Gopackage filesystem

import (
    "os"
    "github.com/SergioLacerda/strategist-skill/internal/ports"
)

type Adapter struct{}

func NewFilesystemAdapter() ports.Filesystem {
    return &Adapter{}
}

func (a *Adapter) ReadFile(path string) ([]byte, error) {
    return os.ReadFile(path)
}

func (a *Adapter) WriteFile(path string, data []byte) error {
    return os.WriteFile(path, data, 0644)
}

2. Makefile Atualizado (World-Class)
makefile# Makefile - Strategist Skill

.PHONY: all build test lint clean wire install release

all: clean wire test lint build

# Build
build:
	go build -o bin/strategist ./cmd/strategist

# Wire (Dependency Injection)
wire:
	wire ./internal

# Test
test:
	go test ./... -v -race -coverprofile=coverage.out
	go tool cover -html=coverage.out -o coverage.html

test-bdd:
	go test ./tests/bdd -v

# Lint
lint:
	golangci-lint run --timeout=5m

# Format
fmt:
	go fmt ./...
	gofumpt -l -w .

# Clean
clean:
	rm -rf bin/ dist/ coverage.*

# Install
install:
	go install ./cmd/strategist

# Release
release:
	goreleaser release --clean

# Bootstrap (segurança)
bootstrap:
	./scripts/bootstrap.sh

# Security
sbom:
	syft . -o cyclonedx > sbom.cdx.json

cosign:
	cosign sign-blob --key cosign.key dist/strategist_linux_amd64

3. .golangci.yaml Atualizado
YAMLrun:
  timeout: 5m
  modules-download-mode: readonly

linters:
  enable:
    - errcheck
    - gosimple
    - govet
    - ineffassign
    - staticcheck
    - typecheck
    - unused
    - gosec
    - revive
    - misspell
    - gofumpt
    - gocyclo
    - dupl
    - goconst

linters-settings:
  gosec:
    excludes:
      - G101  # hardcoded credentials (aceitamos em config)
  gocyclo:
    max-complexity: 15
  dupl:
    threshold: 120

issues:
  exclude-rules:
    - path: _test\.go
      linters:
        - gosec
        - dupl

4. Exemplo de Teste Unitário com Mocks
internal/application/orchestrator_test.go
Gopackage application_test

import (
    "context"
    "testing"
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/mock"
    
    "github.com/SergioLacerda/strategist-skill/internal/application"
    "github.com/SergioLacerda/strategist-skill/internal/domain/mission"
    "github.com/SergioLacerda/strategist-skill/internal/ports"
)

type MockLLM struct { mock.Mock }
func (m *MockLLM) Execute(ctx context.Context, prompt, persona string) (string, error) {
    args := m.Called(ctx, prompt, persona)
    return args.String(0), args.Error(1)
}

type MockRepo struct { mock.Mock }
func (m *MockRepo) Save(ctx context.Context, mis *mission.Mission) error {
    args := m.Called(ctx, mis)
    return args.Error(0)
}

func TestOrchestrator_ExecuteMission(t *testing.T) {
    llm := new(MockLLM)
    repo := new(MockRepo)
    git := /* mock similar */
    fs := /* mock similar */

    orch := application.NewOrchestrator(llm, repo, git, fs)

    spec := mission.Spec{
        Objective: "Implementar autenticação JWT",
    }

    llm.On("Execute", mock.Anything, mock.Anything, mock.Anything).Return("Código gerado...", nil)
    repo.On("Save", mock.Anything, mock.Anything).Return(nil)

    result, err := orch.ExecuteMission(context.Background(), spec)

    assert.NoError(t, err)
    assert.NotNil(t, result)
    llm.AssertExpectations(t)
    repo.AssertExpectations(t)
}   